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
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
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

type TemplateArgs struct {
	StopBlock uint
}

func main() {
	err := godotenv.Load()
	workingDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	outDir := flag.String("out-dir", filepath.Join(workingDir, "docker"), "docker output directory")
	serviceFile := flag.String("service-file", os.Getenv("SERVICE_FILE"), "gcp service file")
	exec := flag.String("defid", "", "defid executable location")
	downloadSnap := flag.Bool("download", false, "download snapshots")
	minHeight := flag.Uint64("min-height", 0, "minimum snapshot height")
	maxHeight := flag.Uint64("max-height", math.MaxUint64, "minimum snapshot height")
	nBlocks := flag.Uint64("nblocks", 50000, "number of block to sync to from snapshot height")
	cli := flag.String("deficli", "", "defid-cli executable location")
	flag.Parse()

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(*serviceFile))
	if err != nil {
		panic(err)
	}
	teamDropBucket := client.Bucket("team-drop")
	relOutDir, err := filepath.Rel(workingDir, *outDir)
	var rootDockerDir string
	if err == nil {
		rootDockerDir = relOutDir
	} else {
		rootDockerDir = *outDir
	}

	err = os.MkdirAll(rootDockerDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	it := teamDropBucket.Objects(ctx, &storage.Query{Prefix: "master-datadir/datadir-", IncludeTrailingDelimiter: false})
	composeFile := NewComposeFile()
	var wg sync.WaitGroup
	var port = 8000

	b, err := os.ReadFile(filepath.Join(workingDir, "Dockerfile.template"))
	if err != nil {
		panic(err)
	}
	template, err := template.New("test").Parse(string(b))
	if err != nil {
		panic(err)
	}

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

		if uint64(startBlock) < *minHeight || uint64(startBlock) > *maxHeight {
			continue
		}
		stopBlock := startBlock + int(*nBlocks) + 5
		buildConfig := NewBuildConfigBuilder().
			Context(fmt.Sprintf("./%s", snapshotName)).
			WithArg("volume_name", snapshotName).
			Build()

		service := Service{
			Build: buildConfig,
		}
		service.Ports = []Port{NewPort(8554, uint(port))}
		service.Deploy = DeployConfig{
			RestartPolicy: RestartPolicy{
				Condition:   "on-failure",
				Delay:       "10s",
				MaxAttempts: 40,
				Window:      "10s",
			},
		}
		port++
		composeFile.AddService(snapshotName, service)
		wg.Add(1)
		ctx = context.WithValue(ctx, "workingDir", workingDir)
		ctx = context.WithValue(ctx, "rootDir", rootDockerDir)

		args := TemplateArgs{StopBlock: uint(stopBlock)}
		go func() {
			defer wg.Done()
			generateDockerContainer(ctx, snapshot, teamDropBucket, template, args, *exec, *cli, *downloadSnap)
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

	println()
	println("Done! next steps..")
	println("	$ cd", rootDockerDir)
	println("	$ docker compose up")

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

func generateDockerContainer(ctx context.Context, snapshot *storage.ObjectAttrs, teamDropBucket *storage.BucketHandle, template *template.Template, args TemplateArgs, exec, cli string, download bool) {
	var workingDir = ctx.Value("workingDir").(string)
	var rootDir = ctx.Value("rootDir").(string)
	snapshotDir := filepath.Join(rootDir, BaseName(snapshot.Name))
	err := os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if download {
		snapshotObj := teamDropBucket.Object(snapshot.Name)
		// Download snapshot, TODO : use aria2 to download snapshots
		snapshotFilePath := filepath.Join(snapshotDir, "snapshot.tar.gz")
		//fileExists, err := Exists(snapshotDir)
		// Prevent Re-download
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
	err = OSCopyFile(exec, filepath.Join(snapshotDir, "defid"))
	if err != nil {
		panic(err)
	}
	err = OSCopyFile(cli, filepath.Join(snapshotDir, "defi-cli"))
	if err != nil {
		panic(err)
	}
	err = OSCopyFile(filepath.Join(workingDir, "start.sh"), filepath.Join(snapshotDir, "start.sh"))
	if err != nil {
		panic(err)
	}
	dockerFilePath := filepath.Join(snapshotDir, "Dockerfile")
	dockerFile, err := os.Create(dockerFilePath)
	defer dockerFile.Close()
	if err != nil {
		panic(err)
	}
	err = template.Execute(dockerFile, args)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Finished downloading %s \n", BaseName(snapshot.Name))
}
