package main

import (
	t "turbo"
)

// TODO support cmd tool
func main() {
	t.LoadServiceConfig("turbo/example/inventoryservice")
	t.GenerateHandler()
}
