package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func downloadTgzFile(link string) error {
	idx := strings.LastIndex(link, "/")
	f, err := os.Create("./charts/" + link[idx:])
	if err != nil {
		return err
	}
	defer f.Close()
	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func getIndexFile(repoURL string) (*repo.IndexFile, error) {
	r, err := repo.NewChartRepository(&repo.Entry{
		URL: "https://" + repoURL,
	}, getter.All(cli.New()))
	if err != nil {
		return nil, fmt.Errorf("error creating repo err=%s", err.Error())
	}

	index, err := r.DownloadIndexFile()
	if err != nil {
		return nil, fmt.Errorf("error downloading index.yaml err=%s", err.Error())
	}
	file, err := repo.LoadIndexFile(index)
	if err != nil {
		return nil, fmt.Errorf("error loading index file, err=%s", err.Error())
	}
	return file, nil
}

func renderHelmChart(tgzDir string) ([]byte, error) {
	out, err := exec.Command("helm", "template", tgzDir).Output()
	if err != nil {
		return nil, err
	}

	return out, nil

	// file, err := os.Open(tgzDir)
	// if err != nil {
	// 	return nil, fmt.Errorf("error opening tgz file")
	// }
	// defer file.Close()
	// files, err := loader.LoadArchiveFiles(file)
	// if err != nil {
	// 	return nil, fmt.Errorf("error loading chart: %w", err)
	// }
	//
	// chart, err := loader.LoadFiles(files)
	// if err != nil {
	// 	return nil, fmt.Errorf("error loading archive, err=%s", err.Error())
	// }
	//
	// for _, value := range chart.Values {
	// 	v, ok := value.(map[string]any)
	// 	if !ok {
	// 		continue
	// 	}
	// 	if v == nil {
	// 		fmt.Println("This one was nil")
	// 		continue
	// 	}
	// 	for k, s := range v {
	// 		fmt.Printf("Key: %v\nValue: %v\n", k, s)
	// 	}
	// }
	//
	// for _, f := range files {
	// 	if f.Name != "values.yaml" {
	// 		continue
	// 	}
	// 	fmt.Println("ASD NUMERO 2")
	// 	val, err := chartutil.ReadValues(f.Data)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	rendered, err := engine.Render(chart, val)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("error rendering err=%s", err.Error())
	// 	}
	// 	for key, value := range rendered {
	// 		fmt.Println(key + "\n" + value)
	// 	}
	// }
	// fmt.Println("out of for loop")
	//
	// return nil, nil
}

func getContainerImages(values []byte) ([]map[string]string, error) {
	valStr := string(values)
	idx := strings.LastIndex(valStr, "deployment.yaml")
	if idx == -1 {
		return nil, fmt.Errorf("no deployment.yaml found")
	}
	deployment := valStr[idx:]
	parts := strings.Split(deployment, "\n")
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed deployment.yaml")
	}
	var infoList []map[string]string
	for _, line := range parts {
		info := make(map[string]string)
		if !strings.Contains(line, "image:") {
			continue
		}
		image := strings.Split(line, "\"")
		if len(image) != 3 {
			return nil, fmt.Errorf("error, malformed image section")
		}
		img := image[1]
		info["name"] = img
		infoList = append(infoList, info)
	}

	return infoList, nil
}

func downloadDockerImages(images []map[string]string) ([]map[string]string, error) {
	dockerPath := "./docker"

	for i, img := range images {
		imageRef := img["name"]
		fmt.Printf("Processing image: %s\n", imageRef)

		ref, err := name.ParseReference(imageRef)
		if err != nil {
			return nil, fmt.Errorf("error parsing reference for image %s: %s", imageRef, err.Error())
		}

		fmt.Printf("Downloading docker image %s...\n", imageRef)
		imgObj, err := remote.Image(ref)
		if err != nil {
			return nil, fmt.Errorf("error downloading image %s: %s", imageRef, err.Error())
		}

		outputPath := fmt.Sprintf("%s/%s.tar", dockerPath, imageRef)
		f, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("error creating file container for %s: %s", imageRef, err.Error())
		}
		defer f.Close()

		fmt.Printf("Saving image %s to disk...\n", imageRef)
		if err := crane.Save(imgObj, ref.Context().Name(), outputPath); err != nil {
			return nil, fmt.Errorf("error saving image %s to disk: %s", imageRef, err.Error())
		}

		size, err := imgObj.Size()
		if err != nil {
			return nil, fmt.Errorf("error getting size for image %s: %s", imageRef, err.Error())
		}

		layers, err := imgObj.Layers()
		if err != nil {
			return nil, fmt.Errorf("error getting layers for image %s: %s", imageRef, err.Error())
		}
		numLayers := len(layers)

		images[i]["size"] = fmt.Sprintf("%d", size)
		images[i]["no. layers"] = fmt.Sprintf("%d", numLayers)

		fmt.Printf("Image %s downloaded: size=%d bytes, layers=%d\n", imageRef, size, numLayers)
	}
	return images, nil
}
