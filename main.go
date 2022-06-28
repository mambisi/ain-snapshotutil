package main

import (
	"bufio"
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func BaseName(filename string) string {
	filename = path.Base(filename)
	var extension = ".tar.gz"
	var name = filename[0 : len(filename)-len(extension)]
	return name
}

func OSCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func main() {

	defidExec := flag.String("defid-exec", "", "defid executable location")
	flag.Parse()

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
		snapshotName := BaseName(snapshot.Name)
		split := strings.Split(snapshotName, "-")
		startBlock, err := strconv.Atoi(split[1])
		if err != nil {
			panic(err)
		}
		stopBlock := startBlock + 50000
		buildConfig := NewBuildConfigBuilder().
			Context(fmt.Sprintf("./%s", snapshotName)).
			WithArg("stop_block", fmt.Sprintf("%v", stopBlock)).
			WithArg("volume_name", snapshotName).
			Build()

		service := Service{
			Build: buildConfig,
		}
		service.Ports = []Port{NewPort(8554, uint(port))}
		service.CustomFields = map[string]interface{}{"restart": "on-failure"}
		port++
		composeFile.AddService(snapshotName, service)
		wg.Add(1)
		go func() {
			defer wg.Done()
			generateDockerfile(snapshot, *defidExec, teamDropBucket, ctx, workingDir, rootDockerDir)
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

func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func generateDockerfile(snapshot *storage.ObjectAttrs, defidExec string, teamDropBucket *storage.BucketHandle, ctx context.Context, workingDir string, rootDir string) {
	snapshotDir := filepath.Join(rootDir, BaseName(snapshot.Name))
	err := os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	snapshotObj := teamDropBucket.Object(snapshot.Name)
	// Download snapshot, TODO : use aria2 to download snapshots
	snapshotFilePath := filepath.Join(snapshotDir, "snapshot.tar.gz")
	fileExists, err := Exists(snapshotDir)
	// Prevent Re-download
	if !fileExists || err != nil {
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
	}
	defidExecPath := filepath.Join(snapshotDir, "defid")
	err = OSCopyFile(defidExec, defidExecPath)
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
	fmt.Printf("Finished downloading %s \n", BaseName(snapshot.Name))
}
