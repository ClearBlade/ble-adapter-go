package ble

import "strings"

func hexMatch(s, pattern string) bool {
	const hexDigits = "0123456789abcdef"
	if len(s) != len(pattern) {
		return false
	}
	for i := range s {
		switch pattern[i] {
		case 'x':
			if strings.IndexByte(hexDigits, s[i]) == -1 {
				return false
			}
		default:
			if s[i] != pattern[i] {
				return false
			}
		}
	}
	return true
}

// ValidUUID checks whether a string is a valid UUID.
func ValidUUID(u string) bool {
	switch len(u) {
	case 4:
		return hexMatch(u, "xxxx")
	case 8:
		return hexMatch(u, "xxxxxxxx")
	case 36:
		return hexMatch(u, "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	default:
		return false
	}
}

// ConvertUUID creates the 128 bit representation of a uuid from a
//16, 32, or 128 bit uuid
func ConvertUUID(uuid string) string {
	var longuuid = BluetoothBaseUUID

	switch len(uuid) {
	case 4:
		//Convert 16bit uuid to 128 bit uuid
		strings.Replace(longuuid, "00000000", "0000"+uuid, 1)
	case 8:
		//convert 32bit uuid to 128 bit uuid
		strings.Replace(longuuid, "00000000", uuid, 1)
	default:
		longuuid = uuid
	}
	return strings.ToLower(longuuid)
}
