package receiver

func chunkData(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		chunks = append(chunks, chunk)
	}

	return chunks
}

func concatenateChunks(chunks [][]byte) []byte {
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	result := make([]byte, 0, totalSize)

	for _, chunk := range chunks {
		result = append(result, chunk...)
	}
	return result
}
