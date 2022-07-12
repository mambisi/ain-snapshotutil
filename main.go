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
	defidExec := flag.String("defid", "", "defid executable location")
	downloadSnap := flag.Bool("download", false, "download snapshots")
	static := flag.Bool("static", false, "download snapshots")
	minHeight := flag.Uint64("min-height", 0, "minimum snapshot height")
	maxHeight := flag.Uint64("max-height", math.MaxUint64, "minimum snapshot height")
	r := flag.String("range", "..", "snapshot range eg. 100000..500000 or specific snapshots eg. 100000,400000,600000")
	nBlocks := flag.Uint64("nblocks", 50000, "number of block to sync to from snapshot height")
	defiCliExec := flag.String("deficli", "", "defid-cli executable location")
	flag.Parse()

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("SERVICE_FILE")))
	if err != nil {
		panic(err)
	}

	pRange, err := ParseRange(*r)
	if err != nil {
		panic(err)
	}

	teamDropBucket := client.Bucket("team-drop")
	rootDockerDir := *outDir
	err = os.MkdirAll(rootDockerDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	it := teamDropBucket.Objects(ctx, &storage.Query{Prefix: "master-datadir/datadir-", IncludeTrailingDelimiter: false})
	composeFile := NewComposeFile()
	var wg sync.WaitGroup
	var port = 3000
	var b []byte
	if *static {
		b, err = os.ReadFile(filepath.Join(workingDir, "DockerfileStatic.template"))
		if err != nil {
			panic(err)
		}
	} else {
		b, err = os.ReadFile(filepath.Join(workingDir, "Dockerfile.template"))
		if err != nil {
			panic(err)
		}
	}

	tmpl, err := template.New("test").Parse(string(b))
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

		if !pRange.InRange(uint64(startBlock)) {
			continue
		}
		var a TemplateArgs

		if !*static {
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
			a = TemplateArgs{StopBlock: uint(stopBlock)}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			generateDockerfile(tmpl, snapshot, *defidExec, *defiCliExec, *static, *downloadSnap, a, teamDropBucket, ctx, workingDir, rootDockerDir)
		}()

	}
	wg.Wait()

	if !*static {
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

func generateDockerfile(tmpl *template.Template, snapshot *storage.ObjectAttrs, defidExec, defiCli string, downloadSnap bool, static bool, args TemplateArgs, teamDropBucket *storage.BucketHandle, ctx context.Context, workingDir string, rootDir string) {
	snapshotDir := filepath.Join(rootDir, BaseName(snapshot.Name))
	err := os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if downloadSnap {
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
	if !static {
		err = OSCopyFile(defidExec, filepath.Join(snapshotDir, "defid"))
		if err != nil {
			panic(err)
		}
		err = OSCopyFile(defiCli, filepath.Join(snapshotDir, "defi-cli"))
		if err != nil {
			panic(err)
		}
		err = OSCopyFile(filepath.Join(workingDir, "start.sh"), filepath.Join(snapshotDir, "start.sh"))
		if err != nil {
			panic(err)
		}
	}
	dockerFilePath := filepath.Join(snapshotDir, "Dockerfile")
	dockerFile, err := os.Create(dockerFilePath)
	defer dockerFile.Close()
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(dockerFile, args)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Finished downloading %s \n", BaseName(snapshot.Name))
}
