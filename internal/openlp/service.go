package openlp

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
)

// ServiceFile represents a complete OpenLP service file.
type ServiceFile struct {
	Items []ServiceItem
}

// WriteOSZ writes the service file as an .osz (ZIP) archive.
func (sf *ServiceFile) WriteOSZ(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Build the JSON array: [openlp_core, serviceitem, serviceitem, ...]
	coreHeader := map[string]interface{}{
		"lite-service":               false,
		"service-theme":              nil,
		"openlp-servicefile-version": 3,
	}
	coreJSON, err := json.Marshal(coreHeader)
	if err != nil {
		return err
	}
	coreRaw := json.RawMessage(coreJSON)

	var entries []json.RawMessage

	// First entry: openlp_core
	entry0, err := json.Marshal(map[string]*json.RawMessage{
		"openlp_core": &coreRaw,
	})
	if err != nil {
		return err
	}
	entries = append(entries, json.RawMessage(entry0))

	// Subsequent entries: service items
	for _, item := range sf.Items {
		wrapper := map[string]interface{}{
			"serviceitem": map[string]interface{}{
				"header": item.Header,
				"data":   item.Data,
			},
		}
		entryJSON, err := json.Marshal(wrapper)
		if err != nil {
			return fmt.Errorf("marshaling service item %q: %w", item.Header.Title, err)
		}
		entries = append(entries, json.RawMessage(entryJSON))
	}

	// Marshal the complete array
	osjData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling service data: %w", err)
	}

	// Write service_data.osj to the ZIP
	w, err := zw.Create("service_data.osj")
	if err != nil {
		return fmt.Errorf("creating osj in zip: %w", err)
	}
	if _, err := w.Write(osjData); err != nil {
		return fmt.Errorf("writing osj data: %w", err)
	}

	return nil
}
