package heuristic

// parseDQTMarkers extracts JPEG quantization tables from DQT segments (0xFFDB).
func parseDQTMarkers(raw []byte) ([][]int, error) {
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return nil, nil
	}

	var tables [][]int
	i := 2
	for i+3 < len(raw) {
		if raw[i] != 0xFF {
			i++
			continue
		}
		marker := raw[i+1]
		if marker == 0xD9 || marker == 0xDA { // EOI / SOS
			break
		}
		if marker == 0x00 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if i+3 >= len(raw) {
			break
		}
		length := int(raw[i+2])<<8 | int(raw[i+3])
		if length < 2 || i+2+length > len(raw) {
			break
		}
		segment := raw[i+4 : i+2+length]
		if marker == 0xDB { // DQT
			pos := 0
			for pos < len(segment) {
				info := segment[pos]
				pos++
				precision := info >> 4
				count := 64
				if precision == 1 {
					count = 128
				}
				if pos+count > len(segment) {
					break
				}
				table := make([]int, 64)
				if precision == 0 {
					for j := 0; j < 64; j++ {
						table[j] = int(segment[pos+j])
					}
					pos += 64
				} else {
					for j := 0; j < 64; j++ {
						table[j] = int(segment[pos])<<8 | int(segment[pos+1])
						pos += 2
					}
				}
				tables = append(tables, table)
			}
		}
		i += 2 + length
	}
	return tables, nil
}

// isGenericExportQuant heuristically detects common non-camera export tables
// (very flat / rounded values typical of re-encoded web exports).
func isGenericExportQuant(tables [][]int) (tool string, ok bool) {
	if len(tables) == 0 {
		return "", false
	}
	t := tables[0]
	uniq := map[int]struct{}{}
	sum := 0
	for _, v := range t {
		uniq[v] = struct{}{}
		sum += v
	}
	avg := float64(sum) / float64(len(t))
	// Generic exports often use few distinct steps and mid-range averages.
	if len(uniq) <= 12 && avg >= 8 && avg <= 40 {
		return "generic-export", true
	}
	// All ones / near-constant → heavy recompression / synthetic export.
	if len(uniq) <= 3 {
		return "flat-quant", true
	}
	return "", false
}
