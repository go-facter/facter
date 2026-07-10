// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// collectDMI builds the structured dmi fact — the SMBIOS/DMI inventory Facter
// exposes as bios / board / chassis sub-maps plus the top-level manufacturer and
// the product (name / serial_number / uuid) map. Sources are per-OS: the Linux
// /sys/class/dmi/id tree, ioreg/sysctl on Darwin, and WMI on Windows. Empty
// members are omitted, exactly as Facter drops facts it cannot resolve, so a bare
// container (no DMI exposed) yields no fact rather than a map of blanks.
func (c *Collection) collectDMI() (any, bool) {
	var d dmiData
	switch c.env.goos {
	case "linux":
		d = c.dmiLinux()
	case "darwin":
		d = c.dmiDarwin()
	case "windows":
		d = c.dmiWindows()
	default:
		return nil, false
	}
	m := d.toMap()
	if len(m) == 0 {
		return nil, false
	}
	return m, true
}

// dmiData is the OS-neutral projection of the DMI inventory the collectors fill.
type dmiData struct {
	biosVendor, biosVersion, biosDate string
	boardVendor, boardProduct         string
	boardSerial, boardAssetTag        string
	chassisType, chassisAssetTag      string
	manufacturer                      string
	productName, productSerial        string
	productUUID                       string
}

// toMap renders the DMI data as Facter's dmi structured fact, omitting any empty
// member and any sub-map that would be wholly empty.
func (d dmiData) toMap() map[string]any {
	out := map[string]any{}
	putNonEmpty(out, "manufacturer", d.manufacturer)

	bios := map[string]any{}
	putNonEmpty(bios, "vendor", d.biosVendor)
	putNonEmpty(bios, "version", d.biosVersion)
	putNonEmpty(bios, "release_date", d.biosDate)
	if len(bios) > 0 {
		out["bios"] = bios
	}

	board := map[string]any{}
	putNonEmpty(board, "manufacturer", d.boardVendor)
	putNonEmpty(board, "product", d.boardProduct)
	putNonEmpty(board, "serial_number", d.boardSerial)
	putNonEmpty(board, "asset_tag", d.boardAssetTag)
	if len(board) > 0 {
		out["board"] = board
	}

	chassis := map[string]any{}
	putNonEmpty(chassis, "type", d.chassisType)
	putNonEmpty(chassis, "asset_tag", d.chassisAssetTag)
	if len(chassis) > 0 {
		out["chassis"] = chassis
	}

	product := map[string]any{}
	putNonEmpty(product, "name", d.productName)
	putNonEmpty(product, "serial_number", d.productSerial)
	putNonEmpty(product, "uuid", d.productUUID)
	if len(product) > 0 {
		out["product"] = product
	}

	return out
}

// putNonEmpty sets m[key]=val only when val is not the empty string.
func putNonEmpty(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}

// dmiLinux reads the /sys/class/dmi/id tree the kernel exposes, mapping the numeric
// chassis_type code to its SMBIOS name.
func (c *Collection) dmiLinux() dmiData {
	read := func(name string) string {
		s, _ := c.env.readText("/sys/class/dmi/id/" + name)
		return strings.TrimSpace(s)
	}
	return dmiData{
		biosVendor:      read("bios_vendor"),
		biosVersion:     read("bios_version"),
		biosDate:        read("bios_date"),
		boardVendor:     read("board_vendor"),
		boardProduct:    read("board_name"),
		boardSerial:     read("board_serial"),
		boardAssetTag:   read("board_asset_tag"),
		chassisType:     chassisTypeName(read("chassis_type")),
		chassisAssetTag: read("chassis_asset_tag"),
		manufacturer:    read("sys_vendor"),
		productName:     read("product_name"),
		productSerial:   read("product_serial"),
		productUUID:     read("product_uuid"),
	}
}

// dmiDarwin fills what a Mac exposes: hw.model as the product name and the
// platform serial/UUID from ioreg, with Apple as the manufacturer.
func (c *Collection) dmiDarwin() dmiData {
	var d dmiData
	if out, ok := c.env.cmd("sysctl", "-n", "hw.model"); ok {
		d.productName = strings.TrimSpace(out)
	}
	if d.productName != "" {
		d.manufacturer = "Apple Inc."
	}
	if out, ok := c.env.cmd("ioreg", "-d2", "-c", "IOPlatformExpertDevice"); ok {
		d.productSerial = ioregValue(out, "IOPlatformSerialNumber")
		d.productUUID = ioregValue(out, "IOPlatformUUID")
	}
	return d
}

// ioregValue pulls the quoted value of key from an ioreg dump line of the form
// "IOPlatformSerialNumber" = "C02XXXXXX".
func ioregValue(out, key string) string {
	needle := `"` + key + `"`
	for _, line := range strings.Split(out, "\n") {
		i := strings.Index(line, needle)
		if i < 0 {
			continue
		}
		rest := line[i+len(needle):]
		j := strings.Index(rest, "= ")
		if j < 0 {
			continue
		}
		return strings.Trim(strings.TrimSpace(rest[j+2:]), `"`)
	}
	return ""
}

// dmiWindows reads the DMI inventory from WMI via three wmic list-format queries.
func (c *Collection) dmiWindows() dmiData {
	var d dmiData
	if out, ok := c.env.cmd("wmic", "bios", "get", "Manufacturer,SMBIOSBIOSVersion,ReleaseDate", "/format:list"); ok {
		kv := parseWmicList(out)
		d.biosVendor = kv["Manufacturer"]
		d.biosVersion = kv["SMBIOSBIOSVersion"]
		d.biosDate = wmicDate(kv["ReleaseDate"])
	}
	if out, ok := c.env.cmd("wmic", "csproduct", "get", "Name,IdentifyingNumber,UUID,Vendor", "/format:list"); ok {
		kv := parseWmicList(out)
		d.productName = kv["Name"]
		d.productSerial = kv["IdentifyingNumber"]
		d.productUUID = kv["UUID"]
		d.manufacturer = kv["Vendor"]
	}
	if out, ok := c.env.cmd("wmic", "baseboard", "get", "Manufacturer,Product,SerialNumber", "/format:list"); ok {
		kv := parseWmicList(out)
		d.boardVendor = kv["Manufacturer"]
		d.boardProduct = kv["Product"]
		d.boardSerial = kv["SerialNumber"]
	}
	return d
}

// parseWmicList parses wmic's /format:list output (blank-separated KEY=VALUE
// blocks) into a single key/value map.
func parseWmicList(out string) map[string]string {
	kv := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return kv
}

// wmicDate converts a WMI CIM datetime (yyyymmdd...) to the mm/dd/yyyy form Facter
// reports, leaving anything shorter untouched.
func wmicDate(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[4:6] + "/" + s[6:8] + "/" + s[0:4]
}

// chassisTypeName maps an SMBIOS chassis-type code to the human name Facter
// reports, passing a non-numeric or unknown value through unchanged.
func chassisTypeName(code string) string {
	code = strings.TrimSpace(code)
	if name, ok := chassisTypes[code]; ok {
		return name
	}
	return code
}

// chassisTypes is the SMBIOS System Enclosure/Chassis type enumeration.
var chassisTypes = map[string]string{
	"1":  "Other",
	"2":  "Unknown",
	"3":  "Desktop",
	"4":  "Low Profile Desktop",
	"5":  "Pizza Box",
	"6":  "Mini Tower",
	"7":  "Tower",
	"8":  "Portable",
	"9":  "Laptop",
	"10": "Notebook",
	"11": "Hand Held",
	"12": "Docking Station",
	"13": "All in One",
	"14": "Sub Notebook",
	"15": "Space-saving",
	"16": "Lunch Box",
	"17": "Main Server Chassis",
	"18": "Expansion Chassis",
	"19": "SubChassis",
	"20": "Bus Expansion Chassis",
	"21": "Peripheral Chassis",
	"22": "RAID Chassis",
	"23": "Rack Mount Chassis",
	"24": "Sealed-case PC",
	"25": "Multi-system chassis",
	"26": "Compact PCI",
	"27": "Advanced TCA",
	"28": "Blade",
	"29": "Blade Enclosure",
	"30": "Tablet",
	"31": "Convertible",
	"32": "Detachable",
	"33": "IoT Gateway",
	"34": "Embedded PC",
	"35": "Mini PC",
	"36": "Stick PC",
}
