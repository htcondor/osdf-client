package stashcp

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func version_status(filePath string) (string, uint64, error) {
	base := path.Base(filePath)
	dir := path.Dir(filePath)

	hash, localSize, err := unique_hash(filePath)
	if err != nil {
		return "", 0, err
	}
	return path.Join(dir, fmt.Sprintf("%s.%s", base, hash)), localSize, nil
}

func generate_destination(filePath string, originPrefix string, shadowOriginPrefix string) (string, uint64, error) {
	hashRaw, localSize, err := version_status(filePath)
	if err != nil {
		return "", 0, err
	}
	hashString := path.Clean(hashRaw)
	cleanedOriginPrefix := path.Clean(originPrefix)
	if strings.HasPrefix(hashString, cleanedOriginPrefix) {
		return shadowOriginPrefix + hashString[len(cleanedOriginPrefix):], localSize, nil
	}
	return "", 0, errors.New("File path must have the origin prefix")
}

func DoShadowIngest(sourceFile string, originPrefix string, shadowOriginPrefix string) (int64, string, error) {
	for idx := 0; idx < 10; idx++ {
		shadowFile, localSize, err := generate_destination(sourceFile, originPrefix, shadowOriginPrefix)
		if err != nil {
			return 0, "", err
		}
		methods := []string{"http"}

		lastRemoteSize := uint64(0)
		lastUpdateTime := time.Now()
		startTime := lastUpdateTime
		maxRuntime := float64(localSize / 10*1024*1024) + 300
		for {
			remoteSize, err := CheckOSDF(shadowFile, methods)
			if err != nil {
				return 0, "", err
			}
			if (localSize == remoteSize) {
				return 0, shadowFile, err
			}

			// If the remote file size is growing, then wait a bit; perhaps someone
			// else is uploading the same file concurrently.
			if duration_s := time.Since(lastUpdateTime).Seconds(); duration_s > 10 {
					// Other uploader is too slow; let's do it ourselves
				if float64(remoteSize - lastRemoteSize) / duration_s < 1024*1024 {
					log.Warnln("Remote uploader is too slow; will do upload from this client")
					break;
				}
				lastRemoteSize = remoteSize
				lastUpdateTime = time.Now()
			}
				// Out of an abundance of caution, never wait more than 10m.
			if time.Since(startTime).Seconds() > maxRuntime {
				log.Warnln("Remote uploader took too long to upload file; will do upload from this client")
				break
			}
			// TODO: Could use a clever backoff scheme here.
			time.Sleep(5)
		}

		uploadBytes, err := DoStashCPSingle(sourceFile, shadowFile, methods, false)

		// See if the file was modified while we were uploading; if not, we'll return success
		shadowFilePost, _, err := generate_destination(sourceFile, originPrefix, shadowOriginPrefix)
		if err != nil {
			return 0, "", err
		}
		if shadowFilePost == shadowFile {	
			return uploadBytes, shadowFile, err
		}
	}
	return 0, "", errors.New("After 10 upload attempts, file was still being modified during ingest.")
}
