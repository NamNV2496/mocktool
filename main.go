package main

import "github.com/namnv2496/mocktool/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
