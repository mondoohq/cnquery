package resources

import (
	"bufio"
	"bytes"
	"embed"
	_ "embed"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
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

func ClientSchema() *runtime.Scheme {
	scheme := runtime.NewScheme()
	// TODO: we need to add more core resources here
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	v1beta1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	policyv1beta1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)
	return scheme
}

func ClientGroups(scheme *runtime.Scheme) (*metav1.APIGroupList, error) {
	vgk := scheme.AllKnownTypes()

	alreadyIncluded := map[string]struct{}{}
	groups := []metav1.APIGroup{}
	for k := range vgk {
		// if group is already added, ignore followup kinds for that group
		_, ok := alreadyIncluded[k.GroupVersion().String()]
		if ok {
			continue
		}

		alreadyIncluded[k.GroupVersion().String()] = struct{}{}

		groups = append(groups,
			metav1.APIGroup{
				Name: k.GroupVersion().Group,
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: k.GroupVersion().String(),
						Version:      k.GroupVersion().Version,
					},
				},
			})
	}

	return &metav1.APIGroupList{
		Groups: groups,
	}, nil
}

//go:embed serverresources/*
var serverresources embed.FS

// CachedServerResources mimics the CachedServerResources call from the dynamic client but based on a manifest file
func CachedServerResources() ([]*metav1.APIResourceList, error) {
	arl := []*metav1.APIResourceList{}
	dir := "serverresources"
	entries, err := serverresources.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		fs := entries[i]
		cachedBytes, err := serverresources.ReadFile(filepath.Join(dir, fs.Name()))
		if err != nil {
			return nil, err
		}
		cachedResources := &metav1.APIResourceList{}
		if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), cachedBytes, cachedResources); err == nil {
			arl = append(arl, cachedResources)
		}
	}

	return arl, nil
}

func ResourcesFromManifest(r io.Reader) ([]k8sRuntime.Object, error) {
	scheme := ClientSchema()
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
				return nil, errors.Wrap(err, "content is not a valid kubernetes manifest")
			}
		}
	}

	return decodedObjects, err
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
