// ディレクトリを暗号化・圧縮してバックアップし、復元するCLIツール。
package main

import (
	"fmt"
	
	"bakashier/constants"
)


func main() {
	fmt.Printf("%s v%s", constants.APP_NAME, constants.APP_VERSION)
}
