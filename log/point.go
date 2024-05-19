package log

import (
	"io/ioutil"
	"text/template"
	"watchlog/log/worker"
	"watchlog/pkg/client/filebeat"
	"watchlog/pkg/ctx"
)

// Run start log pilot
func Run(tmplPath string, baseDir string) error {
	b, err := ioutil.ReadFile(tmplPath)
	if err != nil {
		panic(err)
	}

	c, err := New(string(b), baseDir)
	if err != nil {
		panic(err)
	}

	return worker.NewWorker(c).Run()
}

// New returns a log pilot instance
func New(tplStr string, baseDir string) (*ctx.Context, error) {
	tmpl, err := template.New("pilot").Parse(tplStr)
	if err != nil {
		return nil, err
	}

	p, err := filebeat.NewFilebeatPointer(tmpl, baseDir), nil
	if err != nil {
		return nil, err
	}

	return ctx.NewContext(baseDir, p), nil
}
