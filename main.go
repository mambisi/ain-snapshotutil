package main

import (
	"bufio"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
)

func BaseName(filename string) string {
	filename = path.Base(filename)
	var extension = ".tar.gz"
	var name = filename[0 : len(filename)-len(extension)]
	return name
}

func main() {
	err := godotenv.Load()
	workingDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("SERVICE_FILE")))
	if err != nil {
		panic(err)
	}
	teamDropBucket := client.Bucket("team-drop")
	rootDockerDir := filepath.Join("docker")
	err = os.MkdirAll(rootDockerDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	it := teamDropBucket.Objects(ctx, &storage.Query{Prefix: "master-datadir/datadir-", IncludeTrailingDelimiter: false})
	composeFile := NewComposeFile()
	var wg sync.WaitGroup
	var port = 3000
	for {
		snapshot, err := it.Next()
		if err != nil {
			break
		}
		buildConfig := NewBuildConfigBuilder().
			Context(filepath.Join("./")).
			WithArg("defid_exec", "${DEFID_EXEC}").
			Build()

		service := Service{
			Build: buildConfig,
		}
		service.Ports = []Port{NewPort(8554, uint(port))}
		port++
		composeFile.AddService(BaseName(snapshot.Name), service)
		wg.Add(1)
		go func() {
			defer wg.Done()
			generateDockerfile(snapshot, teamDropBucket, ctx, workingDir, rootDockerDir)
			fmt.Println(workingDir)
		}()

	}
	wg.Wait()
	out, err := yaml.Marshal(composeFile)
	if err != nil {
		panic(err)
	}
	dockerComposeFile, err := os.Create(filepath.Join(rootDockerDir, "docker-compose.yml"))
	defer dockerComposeFile.Close()
	w := bufio.NewWriter(dockerComposeFile)
	_, err = w.Write(out)
	if err != nil {
		panic(err)
	}
	err = w.Flush()
	if err != nil {
		panic(err)
	}
}

func generateDockerfile(snapshot *storage.ObjectAttrs, teamDropBucket *storage.BucketHandle, ctx context.Context, workingDir string, rootDir string) {
	snapshotDir := filepath.Join(rootDir, BaseName(snapshot.Name))
	err := os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	snapshotObj := teamDropBucket.Object(snapshot.Name)
	println(snapshotObj.ObjectName())
	// Download snapshot, TODO : use aria2 to download snapshots
	snapshotFilePath := filepath.Join(snapshotDir, "snapshot.tar.gz")
	snapshotFile, err := os.Create(snapshotFilePath)
	defer snapshotFile.Close()
	if err != nil {
		panic(err)
	}
	reader, err := snapshotObj.NewReader(ctx)
	defer reader.Close()
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(snapshotFile, reader)
	if err != nil {
		panic(err)
	}
	dockerFilePath := filepath.Join(snapshotDir, "Dockerfile")
	f, err := os.Open(filepath.Join(workingDir, "Dockerfile"))
	defer f.Close()
	if err != nil {
		panic(err)
	}
	dockerFile, err := os.Create(dockerFilePath)
	defer dockerFile.Close()
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(dockerFile, f)
	if err != nil {
		panic(err)
	}
}