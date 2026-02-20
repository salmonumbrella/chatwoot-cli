package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestFlagAlias(t *testing.T) {
	t.Run("alias shares value with original", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var val string
		fs.StringVar(&val, "status", "default", "")
		flagAlias(fs, "status", "st")

		if err := fs.Parse([]string{"--st", "open"}); err != nil {
			t.Fatal(err)
		}
		if val != "open" {
			t.Errorf("expected val=open, got %q", val)
		}
	})

	t.Run("alias is hidden", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var val string
		fs.StringVar(&val, "status", "", "")
		flagAlias(fs, "status", "st")

		f := fs.Lookup("st")
		if f == nil {
			t.Fatal("alias not found")
		}
		if !f.Hidden {
			t.Error("alias should be hidden")
		}
	})

	t.Run("flagOrAliasChanged detects alias", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var val string
		cmd.Flags().StringVar(&val, "description", "", "")
		flagAlias(cmd.Flags(), "description", "desc")

		_ = cmd.Flags().Parse([]string{"--desc", "hello"})
		if !flagOrAliasChanged(cmd, "description") {
			t.Error("flagOrAliasChanged should detect alias")
		}
	})

	t.Run("flagOrAliasChanged detects original", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var val string
		cmd.Flags().StringVar(&val, "description", "", "")
		flagAlias(cmd.Flags(), "description", "desc")

		_ = cmd.Flags().Parse([]string{"--description", "hello"})
		if !flagOrAliasChanged(cmd, "description") {
			t.Error("flagOrAliasChanged should detect original")
		}
	})

	t.Run("flagOrAliasChanged false when neither set", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var val string
		cmd.Flags().StringVar(&val, "description", "", "")
		flagAlias(cmd.Flags(), "description", "desc")

		_ = cmd.Flags().Parse([]string{})
		if flagOrAliasChanged(cmd, "description") {
			t.Error("flagOrAliasChanged should be false")
		}
	})

	t.Run("alias does not inherit required annotation", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var val string
		fs.StringVar(&val, "name", "", "")
		// Simulate MarkFlagRequired by adding the annotation
		f := fs.Lookup("name")
		f.Annotations = map[string][]string{
			cobra.BashCompOneRequiredFlag: {"true"},
		}
		flagAlias(fs, "name", "nm")

		alias := fs.Lookup("nm")
		if _, ok := alias.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			t.Error("alias should not have required annotation")
		}
		// Verify original still has it
		orig := fs.Lookup("name")
		if _, ok := orig.Annotations[cobra.BashCompOneRequiredFlag]; !ok {
			t.Error("original should still have required annotation")
		}
	})

	t.Run("alias bridges Changed to canonical flag", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var val string
		fs.StringVar(&val, "query", "", "")
		flagAlias(fs, "query", "sq")

		if err := fs.Parse([]string{"--sq", "hello"}); err != nil {
			t.Fatal(err)
		}
		if val != "hello" {
			t.Errorf("expected val=hello, got %q", val)
		}
		// The canonical flag should be marked Changed
		if !fs.Lookup("query").Changed {
			t.Error("canonical flag should be Changed when alias is used")
		}
	})

	t.Run("alias forwards SliceValue interface", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var vals []string
		fs.StringArrayVar(&vals, "labels", nil, "")
		flagAlias(fs, "labels", "lb")

		alias := fs.Lookup("lb")
		sv, ok := alias.Value.(pflag.SliceValue)
		if !ok {
			t.Fatal("alias of StringArray should implement SliceValue")
		}
		if err := sv.Append("vip"); err != nil {
			t.Fatal(err)
		}
		if err := sv.Append("urgent"); err != nil {
			t.Fatal(err)
		}
		got := sv.GetSlice()
		if len(got) != 2 || got[0] != "vip" || got[1] != "urgent" {
			t.Errorf("expected [vip urgent], got %v", got)
		}
		if err := sv.Replace([]string{"new"}); err != nil {
			t.Fatal(err)
		}
		got = sv.GetSlice()
		if len(got) != 1 || got[0] != "new" {
			t.Errorf("expected [new], got %v", got)
		}
	})

	t.Run("alias of non-slice does not implement SliceValue", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var val string
		fs.StringVar(&val, "name", "", "")
		flagAlias(fs, "name", "nm")

		alias := fs.Lookup("nm")
		if _, ok := alias.Value.(pflag.SliceValue); ok {
			t.Error("alias of String should not implement SliceValue")
		}
	})

	t.Run("panics on missing flag", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for missing flag")
			}
		}()
		flagAlias(fs, "nonexistent", "ne")
	})
}
