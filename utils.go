package version

func trimLastBlankLine(b []byte) (r []byte) {
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case '\r', '\n', ' ':
			continue
		default:
			return b[0:i+1]
		}
	}
	return b
}
