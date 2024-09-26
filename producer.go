package main

import (
	"fmt"
	"sync"
)

func produceRecords(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("hola")
	return
}

