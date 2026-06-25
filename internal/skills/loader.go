package skills

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxFrontmatterBytes = 64 * 1024

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Trigger     string `yaml:"trigger"`
}

func parseFrontmatter(input []byte) (frontmatter, []byte, error) {
	var fm frontmatter

	input = bytes.TrimPrefix(input, []byte("\xef\xbb\xbf"))

	if !bytes.HasPrefix(input, []byte("---\n")) {
		return fm, input, nil
	}

	rest := input[len("---\n"):]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end == -1 {
		return fm, nil, errors.New("frontmatter not closed")
	}

	yamlPart := rest[:end]
	body := rest[end+len("\n---\n"):]

	if err := yaml.Unmarshal(yamlPart, &fm); err != nil {
		return fm, nil, err
	}

	return fm, body, nil
}

func parseFrontmatterFile(path string) (frontmatter, error) {
	var fm frontmatter

	file, err := os.Open(path)
	if err != nil {
		return fm, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	first, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fm, err
	}
	if strings.TrimRight(first, "\r\n") != "---" {
		return fm, nil
	}

	var yamlPart bytes.Buffer
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fm, readErr
		}
		if strings.TrimRight(line, "\r\n") == "---" {
			if err := yaml.Unmarshal(yamlPart.Bytes(), &fm); err != nil {
				return fm, err
			}
			return fm, nil
		}
		if yamlPart.Len()+len(line) > maxFrontmatterBytes {
			return fm, fmt.Errorf("frontmatter exceeds %d bytes", maxFrontmatterBytes)
		}
		yamlPart.WriteString(line)
		if errors.Is(readErr, io.EOF) {
			break
		}
	}

	return fm, errors.New("frontmatter not closed")
}
