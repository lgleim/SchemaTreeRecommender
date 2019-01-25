package schematree

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"
)

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %.2f MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %.2f MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %.2f MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

func countTreeNodes(schema *SchemaTree) {
	var nodeCount uint64
	var countNodes func(node *schemaNode)
	countNodes = func(node *schemaNode) {
		nodeCount++
		// globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RLock()
		for _, child := range node.Children {
			if child != nil {
				countNodes(child)
			}
		}
		// globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RUnlock()
	}

	for true {
		time.Sleep(60 * time.Second)
		nodeCount = 0
		// locking the root node means locking the entire tree for insertions, BUT we should lock every node to avoid race conditions!
		globalNodeLocks[uintptr(unsafe.Pointer(&schema.Root))%lockPrime].RLock()
		countNodes(&schema.Root)
		globalNodeLocks[uintptr(unsafe.Pointer(&schema.Root))%lockPrime].RUnlock()
		fmt.Printf("\nNodeCount: %v\n\n", nodeCount)
	}
}
