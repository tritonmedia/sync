package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"
	"github.com/tritonmedia/sync/pkg/config"
)

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// scanDir returns a map[string]string of complete files in a directory,
// it's output is comparable to ListObjects()
func scanDir(base string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(base,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip directories
			if info.IsDir() {
				return nil
			}

			relBase, err := filepath.Rel(base, path)
			if err != nil {
				return fmt.Errorf("failed to create relative base path: %v", err)
			}

			files = append(files, relBase)

			return nil
		})
	return files, err
}

// downloadObject downloads a minio object
func downloadObject(m *minio.Client, bucket, key, savePath string) error {
	err := os.MkdirAll(filepath.Dir(savePath), 0700)
	if err != nil {
		return err
	}

	meta, err := m.StatObject(bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return err
	}

	o, err := m.GetObject(bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return err
	}

	out, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// set the full length
	bar := pb.Full.Start64(meta.Size)
	barReader := bar.NewProxyReader(o)

	// write from proxy reader into the file
	_, err = io.Copy(out, barReader)
	if err != nil {
		bar.Finish()
		return err
	}

	bar.Finish()
	return nil
}

func main() {
	if strings.ToLower(os.Getenv("SYNC_LOGGER")) == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	conf, err := config.Load()
	if err != nil {
		logrus.Fatalf("failed to read config: %v", err)
	}

	endpoint := conf.S3.Endpoint
	accessKeyID := conf.S3.AccessKey
	secretAccessKey := conf.S3.SecretAccessKey
	bucketName := conf.S3.Bucket
	syncDir := conf.SaveDir
	useSSL := true

	if syncDir == "" {
		logrus.Infof("syncDir not set, using working directory")
		d, err := os.Getwd()
		if err != nil {
			logrus.Fatalf("failed to read working directory: %v", err)
			return
		}

		syncDir = d
	}

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		logrus.Fatalln(err)
		return
	}

	// every 10 seconds
	for {
		logrus.Infof("reading files in '%s'", syncDir)
		localFiles, err := scanDir(syncDir)
		if err != nil {
			log.Fatalf("failed to read local filesystem: %v", err)
			return
		}

		logrus.Infof("listing objects in '%s'", bucketName)

		remoteFiles := make([]string, 0)
		var remoteSize int64

		// Create a done channel to control 'ListObjects' go routine.
		doneCh := make(chan struct{})
		defer close(doneCh)

		objectCh := minioClient.ListObjects(bucketName, "", true, doneCh)
		for object := range objectCh {
			if strings.Contains(object.Key, "minio.sys.tmp") {
				continue
			}

			if object.Err != nil {
				fmt.Println(object.Err)
				return
			}

			// TODO(jaredallard): validate local files
			remoteSize = remoteSize + object.Size
			remoteFiles = append(remoteFiles, object.Key)
		}

		logrus.Infof("remote has %d remote files, size %s", len(remoteFiles), humanize.Bytes(uint64(remoteSize)))

		localDiff := difference(localFiles, remoteFiles)
		if len(localDiff) != 0 {
			logrus.Warnf("found %d local files not in remote", len(localDiff))
			for _, file := range localDiff {
				logrus.Warnf(" ... %s", file)
			}
		}

		remoteDiff := difference(remoteFiles, localFiles)
		if len(remoteDiff) == 0 {
			logrus.Infoln("no new remote files.")
			return
		}

		logrus.Infof("found %d remote files new files", len(remoteDiff))
		for i, file := range remoteDiff {
			total := float64(len(remoteDiff))
			pos := float64(i + 1)
			progress := (pos / total) * 100

			logrus.Printf(
				"downloading '%s' [%d of %d (%s%%)]",
				file,
				int(pos),
				int(total),
				fmt.Sprintf("%.2f", progress),
			)

			localFile := filepath.Join(syncDir, file)
			err := downloadObject(minioClient, bucketName, file, localFile)
			if err != nil {
				logrus.Warnf("failed to download file '%s': %v", file, err)

				if err := os.Remove(localFile); err != nil && !os.IsNotExist(err) {
					logrus.Warnf("failed to cleanup file: %v", err)
				}
			}
		}

		logrus.Println("sleeping for 10 seconds ...")
		time.Sleep(time.Second * 10)
	}
}
