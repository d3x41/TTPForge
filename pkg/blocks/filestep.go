/*
Copyright © 2023-present, Meta Platforms, Inc. and affiliates
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package blocks

import (
	"context"
	"errors"
	"os/exec"

	"github.com/facebookincubator/ttpforge/pkg/logging"
	"github.com/facebookincubator/ttpforge/pkg/outputs"
	"go.uber.org/zap"
)

// FileStep represents a step in a process that consists of a main action,
// a cleanup action, and additional metadata.
type FileStep struct {
	actionDefaults `yaml:",inline"`
	FilePath       string                  `yaml:"file,omitempty"`
	Executor       string                  `yaml:"executor,omitempty"`
	Environment    map[string]string       `yaml:"env,omitempty"`
	Outputs        map[string]outputs.Spec `yaml:"outputs,omitempty"`
	Args           []string                `yaml:"args,omitempty,flow"`
}

// NewFileStep creates a new FileStep instance and returns a pointer to it.
func NewFileStep() *FileStep {
	return &FileStep{}
}

// IsNil checks if the step is nil or empty and returns a boolean value.
func (f *FileStep) IsNil() bool {
	switch {
	case f.FilePath == "":
		return true
	default:
		return false
	}
}

// Validate validates the FileStep. It checks that the
// Act field is valid, and that either FilePath is set with
// a valid file path, or InlineLogic is set with valid code.
//
// If FilePath is set, it ensures that the file exists and retrieves
// its absolute path.
//
// If Executor is not set, it infers the executor based on the file extension.
// It then checks that the executor is in the system path, and if CleanupStep
// is not nil, it validates the cleanup step as well.
// It logs any errors and returns them.
func (f *FileStep) Validate(execCtx TTPExecutionContext) error {
	if f.FilePath == "" {
		err := errors.New("a TTP must include inline logic or path to a file with the logic")
		logging.L().Error(zap.Error(err))
		return err
	}

	// If FilePath is set, ensure that the file exists.
	fullpath, err := FindFilePath(f.FilePath, execCtx.Vars.WorkDir, nil)
	if err != nil {
		logging.L().Error(zap.Error(err))
		return err
	}

	// Retrieve the absolute path to the file.
	f.FilePath, err = FetchAbs(fullpath, execCtx.Vars.WorkDir)
	if err != nil {
		logging.L().Error(zap.Error(err))
		return err
	}

	// Infer executor if it's not set.
	if f.Executor == "" {
		f.Executor = InferExecutor(f.FilePath)
		logging.L().Debugw("executor set via extension", "exec", f.Executor)
	}

	if f.Executor == ExecutorBinary {
		return nil
	}

	if _, err := exec.LookPath(f.Executor); err != nil {
		logging.L().Error(zap.Error(err))
		return err
	}
	logging.L().Debugw("command found in path", "executor", f.Executor)

	return nil
}

// Template takes each applicable field in the step and replaces any template strings with their resolved values.
//
// **Returns:**
//
// error: error if template resolution fails, nil otherwise
func (f *FileStep) Template(execCtx TTPExecutionContext) error {
	var err error
	f.FilePath, err = execCtx.templateStep(f.FilePath)
	if err != nil {
		return err
	}
	f.Executor, err = execCtx.templateStep(f.Executor)
	if err != nil {
		return err
	}
	for index, value := range f.Args {
		f.Args[index], err = execCtx.templateStep(value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Execute runs the step and returns an error if one occurs.
func (f *FileStep) Execute(execCtx TTPExecutionContext) (*ActResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultExecutionTimeout)
	defer cancel()

	executor := NewExecutor(f.Executor, "", f.FilePath, f.Args, f.Environment)
	result, err := executor.Execute(ctx, execCtx)
	if err != nil {
		return nil, err
	}
	result.Outputs, err = outputs.Parse(f.Outputs, result.Stdout)
	// Send stdout to the output variable
	if f.OutputVar != "" {
		execCtx.Vars.StepVars[f.OutputVar] = result.Stdout
	}
	return result, err
}

// Cleanup is a method to establish a link with the Cleanup interface.
// Assumes that the type is the cleanup step and is invoked by
// f.CleanupStep.Cleanup.
func (f *FileStep) Cleanup(execCtx TTPExecutionContext) (*ActResult, error) {
	// TODO: why call Execute on a cleanup??
	return f.Execute(execCtx)
}
