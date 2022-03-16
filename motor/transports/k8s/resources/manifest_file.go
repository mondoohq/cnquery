package resources

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	yamlsplit  = regexp.MustCompile(`(?m)^---\n`)
	whitespace = regexp.MustCompile(`\s*$`)
)

func MergeManifestFiles(filenames []string) (io.Reader, error) {
	// we read multiple files into a single stream so that it behaves like kubectl apply output
	buf := bytes.NewBuffer(nil)
	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}

		io.Copy(buf, f)
		f.Close()
		// poor man's version to concat yaml files
		buf.WriteString("\n---\n")
	}
	return buf, nil
}

func ResourcesFromManifest(r io.Reader) ([]k8sRuntime.Object, error) {
	scheme := runtime.NewScheme()
	// TODO: we need to add more core resources here
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	v1beta1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	decoder := codecs.UniversalDeserializer()

	decodedObjects := []k8sRuntime.Object{}
	rawData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not load manifest")
	}

	// split file content that is concated via ---
	data := string(rawData)
	slices := yamlsplit.Split(data, -1)

	// iterate over each manifest file
	for _, b := range slices {
		m := whitespace.Find([]byte(b))
		if b == string(m) {
			// ignore all whitespace
			continue
		}

		obj, _, err := decoder.Decode([]byte(b), nil, nil)
		if err == nil && obj != nil {
			decodedObjects = append(decodedObjects, obj)
		} else {
			if !PureCommentManifest([]byte(b)) {
				return decodedObjects, errors.Wrap(err, "content is not a valid kubernetes manifest")
			}
		}
	}
	return decodedObjects, nil
}

// Checks if the manifest is only composed of comments
func PureCommentManifest(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			return false
		}
	}
	return true
}
