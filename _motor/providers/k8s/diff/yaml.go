package diff

import (
	"io"

	"github.com/gonvenience/wrap"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	yamlv3 "gopkg.in/yaml.v3"
)

func YamlDiff(out io.Writer, original []byte, fixed []byte) error {

	var origDocuments []*yamlv3.Node
	var err error
	if origDocuments, err = ytbx.LoadDocuments(original); err != nil {
		return wrap.Error(err, "unable to parse data from original")
	}

	from := ytbx.InputFile{
		Location:  "cosmo original",
		Documents: origDocuments,
	}

	var fixedDocuments []*yamlv3.Node
	if fixedDocuments, err = ytbx.LoadDocuments(fixed); err != nil {
		return wrap.Error(err, "unable to parse data from recommendation")
	}

	to := ytbx.InputFile{
		Location:  "cosmo recommendation",
		Documents: fixedDocuments,
	}

	report, err := dyff.CompareInputFiles(from, to)
	if err != nil {
		return wrap.Errorf(err, "failed to compare input files")
	}

	var reportWriter dyff.ReportWriter

	reportWriter = &dyff.HumanReport{
		Report:            report,
		DoNotInspectCerts: false,
		NoTableStyle:      false,
		OmitHeader:        true,
	}

	err = reportWriter.WriteReport(out)
	if err != nil {
		return wrap.Errorf(err, "failed to print report")
	}
	return nil
}
