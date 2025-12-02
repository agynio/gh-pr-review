package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func encodeJSON(cmd *cobra.Command, payload interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
