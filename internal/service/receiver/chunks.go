package receiver

func chunkData(data []byte, numChunks int) [][]byte {
	// Calculate the size of each chunk
	chunkSize := len(data) / numChunks

	// Create a slice to hold the chunks
	chunks := make([][]byte, numChunks)

	// Split the data into equal-sized chunks
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		var end int
		if i == numChunks-1 {
			// Last chunk may be smaller if the data size is not divisible
			end = len(data)
		} else {
			end = (i + 1) * chunkSize
		}
		chunks[i] = data[start:end]
	}

	return chunks
}
