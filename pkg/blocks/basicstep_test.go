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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalBasic(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name: "Simple basic",
			content: `name: test
description: this is a test
steps:
  - name: testinline
    inline: |
      ls
`,
			wantError: false,
		},
		{
			name: "Simple cleanup basic",
			content: `name: test
description: this is a test
steps:
  - name: testinline
    inline: |
      ls
    cleanup:
      name: test_cleanup
      inline: |
        ls -la
  `,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ttps TTP
			err := yaml.Unmarshal([]byte(tc.content), &ttps)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBasicStepExecuteWithOutput(t *testing.T) {

	// prepare step
	content := `name: test_basic_step
inline: echo {\"foo\":{\"bar\":\"baz\"}}
outputs:
  first:
    filters:
    - json_path: foo.bar`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)

	// execute and check result
	result, err := s.Execute(execCtx)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Outputs))
	assert.Equal(t, "baz", result.Outputs["first"], "first output should be correct")
}

func TestBasicStepExecuteWithTemplate(t *testing.T) {
	content := `name: test_basic_step
inline: echo "this is {[{.StepVars.foo}]}"`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	execCtx.Vars.StepVars = map[string]string{
		"foo": "successfully templated",
	}
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)
	err = s.Template(execCtx)
	require.NoError(t, err)
	result, err := s.Execute(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "this is successfully templated\n", result.Stdout, "stdout should be templated")
}

func TestBasicStepRaisesErrorWithMissingTemplateVariable(t *testing.T) {
	content := `name: test_basic_step
inline: echo "this is {[{.StepVars.foo}]}"`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)
	err = s.Template(execCtx)
	require.Error(t, err)
}

func TestBasicStepExecuteOutputsToOutputVar(t *testing.T) {
	content := `name: test_basic_step
inline: echo "bar"
outputvar: foo`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)
	err = s.Template(execCtx)
	require.NoError(t, err)
	_, err = s.Execute(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "bar", execCtx.Vars.StepVars["foo"], "outputvar should be set")
}

func TestBasicStepExecuteOutputsMultilineToOutputVar(t *testing.T) {
	content := `name: test_basic_step
inline: echo -e "line1\nline2\n\nline4\n"
outputvar: foo`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)
	err = s.Template(execCtx)
	require.NoError(t, err)
	_, err = s.Execute(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\n\nline4\n", execCtx.Vars.StepVars["foo"], "outputvar should be set")
}

func TestBasicStepExecuteOutputsToAndOverwritesOutputVar(t *testing.T) {
	content := `name: test_basic_step
inline: echo "bar"
outputvar: foo`
	var s BasicStep
	execCtx := NewTTPExecutionContext()
	execCtx.Vars.StepVars = map[string]string{
		"foo": "overwrite me",
	}
	err := yaml.Unmarshal([]byte(content), &s)
	require.NoError(t, err)
	err = s.Validate(execCtx)
	require.NoError(t, err)
	err = s.Template(execCtx)
	require.NoError(t, err)
	_, err = s.Execute(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "bar", execCtx.Vars.StepVars["foo"], "outputvar should be set")
}
