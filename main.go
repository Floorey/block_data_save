package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Block struct
type Block struct {
	Index      int
	Timestamp  time.Time
	Values     []float64
	Hash       string
	PrevHash   string
	Mean       float64
	Median     float64
	TwoSDLower float64
	TwoSDUpper float64
	Outliers   []float64
	Text       string
}

// Blockchain struct
type Blockchain struct {
	chain []*Block
	mu    sync.Mutex
}

// NewBlockchain creates a new Blockchain
func NewBlockchain() *Blockchain {
	genesisBlock := &Block{
		Index:      0,
		Timestamp:  time.Now(),
		Values:     nil,
		Hash:       "",
		PrevHash:   "",
		Mean:       0.0,
		Median:     0.0,
		TwoSDLower: 0.0,
		TwoSDUpper: 0.0,
		Outliers:   nil,
		Text:       "",
	}
	genesisBlock.Hash = calculateHash(genesisBlock)

	return &Blockchain{
		chain: []*Block{genesisBlock},
	}
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(values []float64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	prevBlock := bc.chain[len(bc.chain)-1]
	newBlock := &Block{
		Index:      prevBlock.Index + 1,
		Timestamp:  time.Now(),
		Values:     values,
		Hash:       "",
		PrevHash:   prevBlock.Hash,
		Mean:       0.0,
		Median:     0.0,
		TwoSDLower: 0.0,
		TwoSDUpper: 0.0,
		Outliers:   nil,
	}
	bc.calculateBlockStats(newBlock)
	bc.markBlocksWithOutliers()
	newBlock.Hash = calculateHash(newBlock)
	bc.chain = append(bc.chain, newBlock)
}

// calculateBlockStats calculates statistics for the values in a block
func (bc *Blockchain) calculateBlockStats(block *Block) {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		block.Mean = calculateMean(block.Values)
	}()

	go func() {
		defer wg.Done()
		block.Median = calculateMedian(block.Values)
	}()

	go func() {
		defer wg.Done()
		block.TwoSDLower, block.TwoSDUpper = calculateTwoSDRange(block.Values)
	}()

	go func() {
		defer wg.Done()
		block.Outliers = calculateOutliers(block.Values, block.TwoSDLower, block.TwoSDUpper)
	}()

	wg.Wait()
}

// calculateHash calculates the hash for a block
func calculateHash(block *Block) string {
	blockData := fmt.Sprintf("%d%d%v%s%f%f%f%f%v", block.Index, block.Timestamp.Unix(), block.Values, block.PrevHash, block.Mean, block.Median, block.TwoSDLower, block.TwoSDUpper, block.Outliers)
	hash := sha256.Sum256([]byte(blockData))
	return hex.EncodeToString(hash[:])
}

// generateValues generates random values every 5 seconds and adds them to the blockchain
func generateValuesAndAddToBlockchain(bc *Blockchain) {
	valuesChan := make(chan []float64, 10)

	go func() {
		for {
			time.Sleep(5 * time.Second)
			var values []float64
			for i := 0; i < 100; i++ {
				value := rand.Float64()
				values = append(values, value)
			}
			valuesChan <- values
		}
	}()
	for values := range valuesChan {
		bc.AddBlock(values)
	}
}

func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
func calculateMedian(values []float64) float64 {
	sort.Float64s(values)
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2.0
	}
	return values[n/2]
}
func calculateTwoSDRange(values []float64) (lowerBound, upperBound float64) {
	mean := calculateMean(values)
	variance := calculateVariance(values, mean)
	stdDev := math.Sqrt(variance)

	lowerBound = mean - (2 * stdDev)
	upperBound = mean + (2 * stdDev)
	return lowerBound, upperBound
}
func calculateOutliers(values []float64, lowerBound, upperBound float64) (outliers []float64) {
	for _, value := range values {
		if value < lowerBound || value > upperBound {
			outliers = append(outliers, value)
		}
	}
	return outliers
}
func calculateVariance(values []float64, mean float64) float64 {
	sumSquaredDiff := 0.0
	for _, value := range values {
		diff := value - mean
		sumSquaredDiff += diff * diff
	}
	return sumSquaredDiff / float64(len(values))
}
func (bc *Blockchain) markBlocksWithOutliers() {
	for _, block := range bc.chain {
		if len(block.Outliers) > 0 {
			block.Hash = "OUTLIER_BLOCK_HASH"
		}
	}
}

func readDataFromExternalSource(filePath string, format string) ([][]float64, error) {
	var data [][]float64

	// Öffne die Datei
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Lese Daten je nach Dateiformat ein
	switch format {
	case "csv":
		// CSV-Datei einlesen
		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}

		// Konvertiere die eingelesenen Daten in float64
		for _, row := range records {
			var floatRow []float64
			for _, valueStr := range row {
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					return nil, err
				}
				floatRow = append(floatRow, value)
			}
			data = append(data, floatRow)
		}

	case "json":
		// JSON-Datei einlesen
		decoder := json.NewDecoder(file)
		err := decoder.Decode(&data)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("Ungültiges Dateiformat: %s", format)
	}

	return data, nil
}

// main function
func main() {
	bc := NewBlockchain()

	go generateValuesAndAddToBlockchain(bc)

	var choice int
	for {
		fmt.Println("Wählen Sie eine Aktion:")
		fmt.Println("1. Aktuelle Werte ausgeben")
		fmt.Println("2. Blockchain anzeigen")
		fmt.Println("3. Blöcke mit Ausreißern ausgeben")
		fmt.Println("4. Daten aus externe Quelle einlesen und hinzufügen")
		fmt.Println("5. Programm beenden")
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			printBlock(bc.chain[len(bc.chain)-1])
		case 2:
			printBlockchain(bc.chain)
		case 3:
			printOutlierBlocks(bc.chain)
		case 4:
			var filePath, format string
			fmt.Println("Geben Sie den Dateipfad der externen Datenquelle ein:")
			fmt.Scanln(&filePath)
			fmt.Println("Geben Sie das Datenformat ein (csv oder json):")
			fmt.Scanln(&format)

			// Daten aus externer Quelle einlesen (ohne die data-Variable zu verwenden)
			_, err := readDataFromExternalSource(filePath, format)
			if err != nil {
				fmt.Println("Fehler beim Einlesen der externen Datenquelle:", err)
				continue
			}

		case 5:
			return

		default:
			fmt.Println("Ungültige Auswahl!")
		}
	}
}

// printBlock prints the values and metadata of a block
func printBlock(block *Block) {
	fmt.Println("Block Meta-Daten:")
	fmt.Printf("Index: %d\n", block.Index)
	fmt.Printf("Zeitstempel: %v\n", block.Timestamp)
	fmt.Printf("Hash: %s\n", block.Hash)
	fmt.Printf("Vorgänger-Hash: %s\n", block.PrevHash)
	fmt.Printf("Mittelwert: %.2f\n", block.Mean)
	fmt.Printf("Median: %.2f\n", block.Median)
	fmt.Printf("2-SD Bereich: %.2f - %.2f\n", block.TwoSDLower, block.TwoSDUpper)
	fmt.Println("Ausreißer:")
	for _, outlier := range block.Outliers {
		fmt.Printf("%.2f ", outlier)
	}
	fmt.Println("\nWerte im aktuellen Block:")
	for _, value := range block.Values {
		fmt.Printf("%.2f ", value)
	}
	fmt.Println()
}

// printBlockchain prints all blocks in the blockchain
func printBlockchain(chain []*Block) {
	fmt.Println("Blockchain:")
	for _, block := range chain {
		printBlock(block)
	}
}

func printOutlierBlocks(chain []*Block) {
	fmt.Println("Blöcke mit Ausreißern:")
	for _, block := range chain {
		if len(block.Outliers) > 0 {
			printBlock(block)
		}
	}
}
