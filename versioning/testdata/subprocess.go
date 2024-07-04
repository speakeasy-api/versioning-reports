package main

import (
	"context"
	"log"
	"os"

	"github.com/speakeasy-api/sdk-gen-config/versioning"
)

func main() {
	ctx := context.Background()
	err := versioning.AddVersionReport(ctx, versioning.VersionReport{
		Key:          "subprocess" + os.Args[1],
		Priority:     2,
		MustGenerate: true,
		PRReport:     os.Args[2],
	})
	if err != nil {
		log.Fatal(err)
	}
}
