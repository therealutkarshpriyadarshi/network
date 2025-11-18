// +build !amd64

package common

// CalculateChecksumSIMD falls back to optimized implementation on non-amd64
func CalculateChecksumSIMD(data []byte) uint16 {
	return CalculateChecksumFast(data)
}

// CalculateChecksumWithPseudoHeaderSIMD falls back on non-amd64
func CalculateChecksumWithPseudoHeaderSIMD(ph *PseudoHeader, data []byte) uint16 {
	return CalculateChecksumWithPseudoHeaderOptimized(ph, data)
}

// UpdateChecksumSIMD falls back on non-amd64
func UpdateChecksumSIMD(oldChecksum uint16, oldData, newData []byte) uint16 {
	return UpdateChecksum(oldChecksum, oldData, newData)
}

// VerifyChecksumSIMD falls back on non-amd64
func VerifyChecksumSIMD(data []byte, expectedChecksum uint16) bool {
	return VerifyChecksum(data, expectedChecksum)
}
