package python

import (
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/resources"
)

func (k *mqlPythonPackage) id() (string, error) {
	return k.Id()
}

func (p *mqlPythonPackage) init(args *resources.Args) (*resources.Args, PythonPackage, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python.package initialization, it must be a string")
		}

		ppd, err := parseMIME(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing python package data: %s", err)
		}

		obj, err := pythonPackageDetailsToResource(p.MotorRuntime, *ppd)
		if err != nil {
			return nil, nil, fmt.Errorf("error translating python metadata into internal resource: %s", err)
		}

		pPkg, ok := obj.(PythonPackage)
		if !ok {
			return nil, nil, fmt.Errorf("unexpectedly unable to convert to python.package type")
		}

		return nil, pPkg, nil

	}
	return nil, nil, fmt.Errorf("unable to initialize ")
}
