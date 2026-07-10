// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

// dmiLinuxFiles is a complete /sys/class/dmi/id fixture for a QEMU guest.
var dmiLinuxFiles = map[string]string{
	"/sys/class/dmi/id/bios_vendor":       "SeaBIOS\n",
	"/sys/class/dmi/id/bios_version":      "1.16.0\n",
	"/sys/class/dmi/id/bios_date":         "04/01/2014\n",
	"/sys/class/dmi/id/board_vendor":      "Intel\n",
	"/sys/class/dmi/id/board_name":        "440BX\n",
	"/sys/class/dmi/id/board_serial":      "BSN123\n",
	"/sys/class/dmi/id/board_asset_tag":   "BAT456\n",
	"/sys/class/dmi/id/chassis_type":      "10\n",
	"/sys/class/dmi/id/chassis_asset_tag": "CAT789\n",
	"/sys/class/dmi/id/sys_vendor":        "QEMU\n",
	"/sys/class/dmi/id/product_name":      "Standard PC\n",
	"/sys/class/dmi/id/product_serial":    "PSN000\n",
	"/sys/class/dmi/id/product_uuid":      "abcd-uuid\n",
}

func TestDMILinux(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: dmiLinuxFiles}).collection()
	cases := map[string]string{
		"dmi.bios.vendor":           "SeaBIOS",
		"dmi.bios.version":          "1.16.0",
		"dmi.bios.release_date":     "04/01/2014",
		"dmi.board.manufacturer":    "Intel",
		"dmi.board.product":         "440BX",
		"dmi.board.serial_number":   "BSN123",
		"dmi.board.asset_tag":       "BAT456",
		"dmi.chassis.type":          "Notebook",
		"dmi.chassis.asset_tag":     "CAT789",
		"dmi.manufacturer":          "QEMU",
		"dmi.product.name":          "Standard PC",
		"dmi.product.serial_number": "PSN000",
		"dmi.product.uuid":          "abcd-uuid",
	}
	for path, want := range cases {
		if v, ok := c.Value(path); !ok || v != want {
			t.Errorf("%s = %v (ok=%v), want %q", path, v, ok, want)
		}
	}
	// Legacy flat aliases resolve through the structured fact.
	legacy := map[string]string{
		"bios_vendor":       "SeaBIOS",
		"chassistype":       "Notebook",
		"manufacturer":      "QEMU",
		"productname":       "Standard PC",
		"serialnumber":      "PSN000",
		"uuid":              "abcd-uuid",
		"boardserialnumber": "BSN123",
	}
	for name, want := range legacy {
		if v, _ := c.Value(name); v != want {
			t.Errorf("legacy %s = %v, want %q", name, v, want)
		}
	}
}

func TestDMILinuxAbsent(t *testing.T) {
	// A container with no DMI files: the fact is absent, not a map of blanks.
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("dmi"); ok {
		t.Fatal("expected dmi absent with no /sys/class/dmi/id")
	}
}

func TestDMIDarwin(t *testing.T) {
	ioreg := `+-o Root  <class IORegistryEntry>
    "IOPlatformSerialNumber" = "C02XYZ123"
    "IOPlatformUUID" = "1234-5678"
`
	c := (fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.model":                  "MacBookPro18,3\n",
		"ioreg -d2 -c IOPlatformExpertDevice": ioreg,
	}}).collection()
	if v, _ := c.Value("dmi.product.name"); v != "MacBookPro18,3" {
		t.Errorf("product.name = %v", v)
	}
	if v, _ := c.Value("dmi.manufacturer"); v != "Apple Inc." {
		t.Errorf("manufacturer = %v", v)
	}
	if v, _ := c.Value("dmi.product.serial_number"); v != "C02XYZ123" {
		t.Errorf("serial = %v", v)
	}
	if v, _ := c.Value("dmi.product.uuid"); v != "1234-5678" {
		t.Errorf("uuid = %v", v)
	}
}

func TestDMIDarwinNoModel(t *testing.T) {
	// No hw.model -> no manufacturer set, no ioreg -> absent entirely.
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("dmi"); ok {
		t.Fatal("expected dmi absent on darwin without sysctl/ioreg")
	}
}

func TestDMIWindows(t *testing.T) {
	c := (fakeEnv{goos: "windows", cmds: map[string]string{
		"wmic bios get Manufacturer,SMBIOSBIOSVersion,ReleaseDate /format:list": "Manufacturer=American Megatrends\r\nSMBIOSBIOSVersion=P1.40\r\nReleaseDate=20200115000000.000000+000\r\n",
		"wmic csproduct get Name,IdentifyingNumber,UUID,Vendor /format:list":    "Name=To Be Filled\r\nIdentifyingNumber=SN99\r\nUUID=UU-99\r\nVendor=ASUS\r\n",
		"wmic baseboard get Manufacturer,Product,SerialNumber /format:list":     "Manufacturer=ASUSTeK\r\nProduct=PRIME\r\nSerialNumber=BSN99\r\n",
	}}).collection()
	if v, _ := c.Value("dmi.bios.vendor"); v != "American Megatrends" {
		t.Errorf("bios vendor = %v", v)
	}
	if v, _ := c.Value("dmi.bios.release_date"); v != "01/15/2020" {
		t.Errorf("bios date = %v", v)
	}
	if v, _ := c.Value("dmi.product.serial_number"); v != "SN99" {
		t.Errorf("product serial = %v", v)
	}
	if v, _ := c.Value("dmi.manufacturer"); v != "ASUS" {
		t.Errorf("manufacturer = %v", v)
	}
	if v, _ := c.Value("dmi.board.product"); v != "PRIME" {
		t.Errorf("board product = %v", v)
	}
}

func TestDMIGenericAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "plan9"}).collection().Value("dmi"); ok {
		t.Fatal("expected dmi absent on unsupported OS")
	}
}

func TestChassisTypeName(t *testing.T) {
	if got := chassisTypeName("3"); got != "Desktop" {
		t.Errorf("chassis 3 = %q", got)
	}
	if got := chassisTypeName("999"); got != "999" {
		t.Errorf("unknown code passthrough = %q", got)
	}
	if got := chassisTypeName(""); got != "" {
		t.Errorf("empty = %q", got)
	}
}

func TestIoregValue(t *testing.T) {
	out := `"IOPlatformSerialNumber" = "ABC"`
	if got := ioregValue(out, "IOPlatformSerialNumber"); got != "ABC" {
		t.Errorf("found = %q", got)
	}
	if got := ioregValue(out, "Nope"); got != "" {
		t.Errorf("missing key = %q", got)
	}
	if got := ioregValue(`"Key" no-equals`, "Key"); got != "" {
		t.Errorf("no equals = %q", got)
	}
}

func TestParseWmicList(t *testing.T) {
	kv := parseWmicList("A=1\r\n\r\nB=2\r\nnoequals\r\n")
	if kv["A"] != "1" || kv["B"] != "2" {
		t.Errorf("parsed = %v", kv)
	}
	if _, ok := kv["noequals"]; ok {
		t.Error("line without = should be skipped")
	}
}

func TestWmicDate(t *testing.T) {
	if got := wmicDate("20200115000000.000000+000"); got != "01/15/2020" {
		t.Errorf("date = %q", got)
	}
	if got := wmicDate("short"); got != "short" {
		t.Errorf("short passthrough = %q", got)
	}
}
