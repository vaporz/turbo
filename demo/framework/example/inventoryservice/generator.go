package main

import (
	f "zx/demo/framework"
)

// TODO support cmd tool
func main() {
	f.LoadServiceConfig("zx/demo/framework/example/inventoryservice")
	f.GenerateHandler()
}
