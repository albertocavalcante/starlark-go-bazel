package ctx

import (
	"testing"

	"go.starlark.net/starlark"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

func TestNewCtx(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:         label,
		WorkspaceName: "test_workspace",
		BinDir:        "bazel-out/k8-fastbuild/bin",
		GenfilesDir:   "bazel-out/k8-fastbuild/genfiles",
		BuildFilePath: "pkg/BUILD",
	})

	if ctx.Label().String() != "//pkg:target" {
		t.Errorf("expected label //pkg:target, got %s", ctx.Label().String())
	}

	if ctx.Type() != "ctx" {
		t.Errorf("expected type ctx, got %s", ctx.Type())
	}

	if ctx.Truth() != true {
		t.Error("expected ctx to be truthy")
	}
}

func TestCtxLabel(t *testing.T) {
	label, _ := types.ParseLabel("//foo/bar:baz")
	ctx := NewCtx(CtxConfig{Label: label})

	v, err := ctx.Attr("label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l, ok := v.(*types.Label)
	if !ok {
		t.Fatalf("expected *types.Label, got %T", v)
	}

	if l.String() != "//foo/bar:baz" {
		t.Errorf("expected //foo/bar:baz, got %s", l.String())
	}
}

func TestCtxWorkspaceName(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:         label,
		WorkspaceName: "my_workspace",
	})

	v, err := ctx.Attr("workspace_name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, ok := v.(starlark.String)
	if !ok {
		t.Fatalf("expected String, got %T", v)
	}

	if string(s) != "my_workspace" {
		t.Errorf("expected my_workspace, got %s", s)
	}
}

func TestCtxBinDir(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	v, err := ctx.Attr("bin_dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root, ok := v.(*FileRoot)
	if !ok {
		t.Fatalf("expected *FileRoot, got %T", v)
	}

	if root.Path() != "bazel-out/bin" {
		t.Errorf("expected bazel-out/bin, got %s", root.Path())
	}
}

func TestCtxFeatures(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:    label,
		Features: []string{"feature1", "feature2"},
	})

	v, err := ctx.Attr("features")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, ok := v.(*starlark.List)
	if !ok {
		t.Fatalf("expected *starlark.List, got %T", v)
	}

	if list.Len() != 2 {
		t.Errorf("expected 2 features, got %d", list.Len())
	}
}

func TestCtxActions(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	v, err := ctx.Attr("actions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions, ok := v.(*Actions)
	if !ok {
		t.Fatalf("expected *Actions, got %T", v)
	}

	if actions.Type() != "actions" {
		t.Errorf("expected type actions, got %s", actions.Type())
	}
}

func TestCtxAttr(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	// Set some attribute values
	ctx.AttrProxy().Set("name", starlark.String("test"))
	ctx.AttrProxy().Set("srcs", starlark.NewList(nil))

	v, err := ctx.Attr("attr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attr, ok := v.(*AttrProxy)
	if !ok {
		t.Fatalf("expected *AttrProxy, got %T", v)
	}

	nameVal, err := attr.Attr("name")
	if err != nil {
		t.Fatalf("unexpected error getting name: %v", err)
	}

	if string(nameVal.(starlark.String)) != "test" {
		t.Errorf("expected name=test, got %v", nameVal)
	}
}

func TestActionsDeclareFile(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	declareFile, _ := ctx.actions.Attr("declare_file")
	builtin := declareFile.(*starlark.Builtin)

	result, err := builtin.CallInternal(thread, starlark.Tuple{starlark.String("output.txt")}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	file, ok := result.(*File)
	if !ok {
		t.Fatalf("expected *File, got %T", result)
	}

	if file.Basename() != "output.txt" {
		t.Errorf("expected basename output.txt, got %s", file.Basename())
	}
}

func TestActionsDeclareDirectory(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	declareDir, _ := ctx.actions.Attr("declare_directory")
	builtin := declareDir.(*starlark.Builtin)

	result, err := builtin.CallInternal(thread, starlark.Tuple{starlark.String("output_dir")}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	file, ok := result.(*File)
	if !ok {
		t.Fatalf("expected *File, got %T", result)
	}

	if !file.IsDirectory() {
		t.Error("expected directory, got regular file")
	}
}

func TestActionsWrite(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	// First declare a file
	declareFile, _ := ctx.actions.Attr("declare_file")
	fileBuiltin := declareFile.(*starlark.Builtin)
	outputFile, _ := fileBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("test.txt")}, nil)

	// Then write to it
	write, _ := ctx.actions.Attr("write")
	writeBuiltin := write.(*starlark.Builtin)
	_, err := writeBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("output"), outputFile},
		{starlark.String("content"), starlark.String("hello world")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions := ctx.actions.DeclaredActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Type != ActionTypeWrite {
		t.Errorf("expected write action, got %s", actions[0].Type)
	}

	if actions[0].Content != "hello world" {
		t.Errorf("expected content 'hello world', got %s", actions[0].Content)
	}
}

func TestActionsRun(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	// Declare output file
	declareFile, _ := ctx.actions.Attr("declare_file")
	fileBuiltin := declareFile.(*starlark.Builtin)
	outputFile, _ := fileBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("out.txt")}, nil)

	// Create run action
	run, _ := ctx.actions.Attr("run")
	runBuiltin := run.(*starlark.Builtin)
	_, err := runBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("outputs"), starlark.NewList([]starlark.Value{outputFile})},
		{starlark.String("executable"), starlark.String("/bin/echo")},
		{starlark.String("arguments"), starlark.NewList([]starlark.Value{starlark.String("hello")})},
		{starlark.String("mnemonic"), starlark.String("Echo")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions := ctx.actions.DeclaredActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Type != ActionTypeRun {
		t.Errorf("expected run action, got %s", actions[0].Type)
	}

	if actions[0].Mnemonic != "Echo" {
		t.Errorf("expected mnemonic Echo, got %s", actions[0].Mnemonic)
	}

	if actions[0].ExecutableString != "/bin/echo" {
		t.Errorf("expected executable /bin/echo, got %s", actions[0].ExecutableString)
	}
}

func TestActionsRunShell(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	// Declare output file
	declareFile, _ := ctx.actions.Attr("declare_file")
	fileBuiltin := declareFile.(*starlark.Builtin)
	outputFile, _ := fileBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("out.txt")}, nil)

	// Create run_shell action
	runShell, _ := ctx.actions.Attr("run_shell")
	runShellBuiltin := runShell.(*starlark.Builtin)
	_, err := runShellBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("outputs"), starlark.NewList([]starlark.Value{outputFile})},
		{starlark.String("command"), starlark.String("echo hello > $1")},
		{starlark.String("mnemonic"), starlark.String("ShellCmd")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions := ctx.actions.DeclaredActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Type != ActionTypeRunShell {
		t.Errorf("expected run_shell action, got %s", actions[0].Type)
	}

	if actions[0].Command != "echo hello > $1" {
		t.Errorf("expected command 'echo hello > $1', got %s", actions[0].Command)
	}
}

func TestActionsSymlink(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	// Declare source file
	declareFile, _ := ctx.actions.Attr("declare_file")
	fileBuiltin := declareFile.(*starlark.Builtin)
	sourceFile, _ := fileBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("source.txt")}, nil)

	// Declare output file
	outputFile, _ := fileBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("link.txt")}, nil)

	// Create symlink action
	symlink, _ := ctx.actions.Attr("symlink")
	symlinkBuiltin := symlink.(*starlark.Builtin)
	_, err := symlinkBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("output"), outputFile},
		{starlark.String("target_file"), sourceFile},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions := ctx.actions.DeclaredActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Type != ActionTypeSymlink {
		t.Errorf("expected symlink action, got %s", actions[0].Type)
	}
}

func TestActionsExpandTemplate(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label:  label,
		BinDir: "bazel-out/bin",
	})

	thread := &starlark.Thread{}

	// Create template and output files
	templateFile := NewFile("template.txt", "", true)
	outputFile := NewDeclaredFile("pkg/output.txt", "bazel-out/bin")

	// Create expand_template action
	expandTemplate, _ := ctx.actions.Attr("expand_template")
	expandTemplateBuiltin := expandTemplate.(*starlark.Builtin)

	subs := starlark.NewDict(1)
	_ = subs.SetKey(starlark.String("{NAME}"), starlark.String("test"))

	_, err := expandTemplateBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("template"), templateFile},
		{starlark.String("output"), outputFile},
		{starlark.String("substitutions"), subs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actions := ctx.actions.DeclaredActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Type != ActionTypeExpandTemplate {
		t.Errorf("expected expand_template action, got %s", actions[0].Type)
	}
}

func TestActionsArgs(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	thread := &starlark.Thread{}

	argsMethod, _ := ctx.actions.Attr("args")
	argsBuiltin := argsMethod.(*starlark.Builtin)

	result, err := argsBuiltin.CallInternal(thread, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args, ok := result.(*Args)
	if !ok {
		t.Fatalf("expected *Args, got %T", result)
	}

	// Test add method
	addMethod, _ := args.Attr("add")
	addBuiltin := addMethod.(*starlark.Builtin)
	_, err = addBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("--flag"), starlark.String("value")}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values := args.Values()
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}

	if values[0] != "--flag" {
		t.Errorf("expected --flag, got %s", values[0])
	}
}

func TestCtxRunfiles(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	thread := &starlark.Thread{}

	runfilesMethod, _ := ctx.Attr("runfiles")
	runfilesBuiltin := runfilesMethod.(*starlark.Builtin)

	file := NewFile("test.txt", "", true)
	result, err := runfilesBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("files"), starlark.NewList([]starlark.Value{file})},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rf, ok := result.(*Runfiles)
	if !ok {
		t.Fatalf("expected *Runfiles, got %T", result)
	}

	if rf.Type() != "runfiles" {
		t.Errorf("expected type runfiles, got %s", rf.Type())
	}
}

func TestCtxExpandLocation(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	// Set up label map
	depLabel, _ := types.ParseLabel("//dep:lib")
	depFile := NewFile("dep/lib.txt", "bazel-out/bin", false)
	ctx.SetLabelMap(map[string][]*File{
		"//dep:lib": {depFile},
	})

	thread := &starlark.Thread{}

	expandMethod, _ := ctx.Attr("expand_location")
	expandBuiltin := expandMethod.(*starlark.Builtin)

	result, err := expandBuiltin.CallInternal(thread, starlark.Tuple{
		starlark.String("path: $(location //dep:lib)"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(result.(starlark.String))
	expected := "path: bazel-out/bin/dep/lib.txt"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}

	// Test with targets parameter
	targetProxy := NewTargetProxy(depLabel)
	targetProxy.SetFiles([]*File{depFile})

	result2, err := expandBuiltin.CallInternal(thread, nil, []starlark.Tuple{
		{starlark.String("input"), starlark.String("file: $(location //dep:lib)")},
		{starlark.String("targets"), starlark.NewList([]starlark.Value{targetProxy})},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s2 := string(result2.(starlark.String))
	expected2 := "file: bazel-out/bin/dep/lib.txt"
	if s2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, s2)
	}
}

func TestCtxExpandMakeVariables(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{
		Label: label,
		MakeVariables: map[string]string{
			"CC": "/usr/bin/gcc",
		},
	})

	thread := &starlark.Thread{}

	expandMethod, _ := ctx.Attr("expand_make_variables")
	expandBuiltin := expandMethod.(*starlark.Builtin)

	additionalSubs := starlark.NewDict(1)
	_ = additionalSubs.SetKey(starlark.String("MY_VAR"), starlark.String("custom"))

	result, err := expandBuiltin.CallInternal(thread, starlark.Tuple{
		starlark.String("cmd"),
		starlark.String("$(CC) $(MY_VAR) $$HOME"),
		additionalSubs,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(result.(starlark.String))
	expected := "/usr/bin/gcc custom $HOME"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestCtxTokenize(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	thread := &starlark.Thread{}

	tokenizeMethod, _ := ctx.Attr("tokenize")
	tokenizeBuiltin := tokenizeMethod.(*starlark.Builtin)

	result, err := tokenizeBuiltin.CallInternal(thread, starlark.Tuple{
		starlark.String(`-flag "quoted value" plain`),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := result.(*starlark.List)
	if list.Len() != 3 {
		t.Fatalf("expected 3 tokens, got %d", list.Len())
	}

	if string(list.Index(0).(starlark.String)) != "-flag" {
		t.Errorf("expected first token -flag, got %s", list.Index(0))
	}

	if string(list.Index(1).(starlark.String)) != "quoted value" {
		t.Errorf("expected second token 'quoted value', got %s", list.Index(1))
	}
}

func TestCtxPackageRelativeLabel(t *testing.T) {
	label, _ := types.ParseLabel("//pkg/sub:target")
	ctx := NewCtx(CtxConfig{Label: label})

	thread := &starlark.Thread{}

	prlMethod, _ := ctx.Attr("package_relative_label")
	prlBuiltin := prlMethod.(*starlark.Builtin)

	// Test with relative label
	result, err := prlBuiltin.CallInternal(thread, starlark.Tuple{starlark.String(":other")}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l := result.(*types.Label)
	if l.String() != "//pkg/sub:other" {
		t.Errorf("expected //pkg/sub:other, got %s", l.String())
	}

	// Test with absolute label
	result2, err := prlBuiltin.CallInternal(thread, starlark.Tuple{starlark.String("//other:target")}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l2 := result2.(*types.Label)
	if l2.String() != "//other:target" {
		t.Errorf("expected //other:target, got %s", l2.String())
	}

	// Test with existing Label
	existingLabel, _ := types.ParseLabel("//existing:label")
	result3, err := prlBuiltin.CallInternal(thread, starlark.Tuple{existingLabel}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result3 != existingLabel {
		t.Error("expected same label to be returned")
	}
}

func TestFileAttributes(t *testing.T) {
	f := NewFile("pkg/subdir/file.txt", "bazel-out/bin", false)

	// Test basename
	basename, _ := f.Attr("basename")
	if string(basename.(starlark.String)) != "file.txt" {
		t.Errorf("expected basename file.txt, got %s", basename)
	}

	// Test extension
	ext, _ := f.Attr("extension")
	if string(ext.(starlark.String)) != "txt" {
		t.Errorf("expected extension txt, got %s", ext)
	}

	// Test path
	path, _ := f.Attr("path")
	if string(path.(starlark.String)) != "bazel-out/bin/pkg/subdir/file.txt" {
		t.Errorf("expected path bazel-out/bin/pkg/subdir/file.txt, got %s", path)
	}

	// Test is_source
	isSource, _ := f.Attr("is_source")
	if bool(isSource.(starlark.Bool)) != false {
		t.Error("expected is_source false")
	}

	// Test is_directory
	isDir, _ := f.Attr("is_directory")
	if bool(isDir.(starlark.Bool)) != false {
		t.Error("expected is_directory false")
	}

	// Test root
	root, _ := f.Attr("root")
	fileRoot := root.(*FileRoot)
	if fileRoot.Path() != "bazel-out/bin" {
		t.Errorf("expected root bazel-out/bin, got %s", fileRoot.Path())
	}
}

func TestOutputsProxy(t *testing.T) {
	outputs := NewOutputsProxy(true)

	// Set an output
	outFile := NewDeclaredFile("pkg/out.txt", "bazel-out/bin")
	outputs.Set("myout", outFile)

	// Get the output
	v, err := outputs.Attr("myout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != outFile {
		t.Error("expected same file object")
	}

	// Test executable output
	exeFile := NewDeclaredFile("pkg/exe", "bazel-out/bin")
	outputs.SetExecutable(exeFile)

	exeV, err := outputs.Attr("executable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exeV != exeFile {
		t.Error("expected executable file")
	}

	// Test non-existent output
	_, err = outputs.Attr("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent output")
	}
}

func TestAttrNames(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	names := ctx.AttrNames()
	if len(names) == 0 {
		t.Error("expected non-empty attr names")
	}

	// Check some expected names
	expected := []string{"label", "attr", "actions", "bin_dir", "outputs"}
	for _, e := range expected {
		found := false
		for _, n := range names {
			if n == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected attr name %q not found", e)
		}
	}
}

func TestCtxString(t *testing.T) {
	label, _ := types.ParseLabel("//pkg:target")
	ctx := NewCtx(CtxConfig{Label: label})

	s := ctx.String()
	expected := "<rule context for //pkg:target>"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}

	// Test aspect context
	aspectCtx := NewCtx(CtxConfig{Label: label, IsForAspect: true})
	as := aspectCtx.String()
	expectedAspect := "<aspect context for //pkg:target>"
	if as != expectedAspect {
		t.Errorf("expected %q, got %q", expectedAspect, as)
	}
}
