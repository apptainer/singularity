/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package umoci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/apex/log"
	"github.com/docker/go-units"
	"github.com/openSUSE/umoci/oci/casext"
	igen "github.com/openSUSE/umoci/oci/config/generate"
	"github.com/openSUSE/umoci/oci/layer"
	"github.com/openSUSE/umoci/pkg/idtools"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/vbatts/go-mtree"
	"golang.org/x/net/context"
)

// FIXME: This should be moved to a library. Too much of this code is in the
//        cmd/... code, but should really be refactored to the point where it
//        can be useful to other people. This is _particularly_ true for the
//        code which repacks images (the changes to the config, manifest and
//        CAS should be made into a library).

// MtreeKeywords is the set of keywords used by umoci for verification and diff
// generation of a bundle. This is based on mtree.DefaultKeywords, but is
// hardcoded here to ensure that vendor changes don't mess things up.
var MtreeKeywords = []mtree.Keyword{
	"size",
	"type",
	"uid",
	"gid",
	"mode",
	"link",
	"nlink",
	"tar_time",
	"sha256digest",
	"xattr",
}

// MetaName is the name of umoci's metadata file that is stored in all
// bundles extracted by umoci.
const MetaName = "umoci.json"

// MetaVersion is the version of Meta supported by this code. The
// value is only bumped for updates which are not backwards compatible.
const MetaVersion = "2"

// Meta represents metadata about how umoci unpacked an image to a bundle
// and other similar information. It is used to keep track of information that
// is required when repacking an image and other similar bundle information.
type Meta struct {
	// Version is the version of umoci used to unpack the bundle. This is used
	// to future-proof the umoci.json information.
	Version string `json:"umoci_version"`

	// From is a copy of the descriptor pointing to the image manifest that was
	// used to unpack the bundle. Essentially it's a resolved form of the
	// --image argument to umoci-unpack(1).
	From casext.DescriptorPath `json:"from_descriptor_path"`

	// MapOptions is the parsed version of --uid-map, --gid-map and --rootless
	// arguments to umoci-unpack(1). While all of these options technically do
	// not need to be the same for corresponding umoci-unpack(1) and
	// umoci-repack(1) calls, changing them is not recommended and so the
	// default should be that they are the same.
	MapOptions layer.MapOptions `json:"map_options"`
}

// WriteTo writes a JSON-serialised version of Meta to the given io.Writer.
func (m Meta) WriteTo(w io.Writer) (int64, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(io.MultiWriter(buf, w)).Encode(m)
	return int64(buf.Len()), err
}

// WriteBundleMeta writes an umoci.json file to the given bundle path.
func WriteBundleMeta(bundle string, meta Meta) error {
	fh, err := os.Create(filepath.Join(bundle, MetaName))
	if err != nil {
		return errors.Wrap(err, "create metadata")
	}
	defer fh.Close()

	_, err = meta.WriteTo(fh)
	return errors.Wrap(err, "write metadata")
}

// ReadBundleMeta reads and parses the umoci.json file from a given bundle path.
func ReadBundleMeta(bundle string) (Meta, error) {
	var meta Meta

	fh, err := os.Open(filepath.Join(bundle, MetaName))
	if err != nil {
		return meta, errors.Wrap(err, "open metadata")
	}
	defer fh.Close()

	err = json.NewDecoder(fh).Decode(&meta)
	if meta.Version != MetaVersion {
		if err == nil {
			err = fmt.Errorf("unsupported umoci.json version: %s", meta.Version)
		}
	}
	return meta, errors.Wrap(err, "decode metadata")
}

// ManifestStat has information about a given OCI manifest.
// TODO: Implement support for manifest lists, this should also be able to
//       contain stat information for a list of manifests.
type ManifestStat struct {
	// TODO: Flesh this out. Currently it's only really being used to get an
	//       equivalent of docker-history(1). We really need to add more
	//       information about it.

	// History stores the history information for the manifest.
	History []historyStat `json:"history"`
}

// Format formats a ManifestStat using the default formatting, and writes the
// result to the given writer.
// TODO: This should really be implemented in a way that allows for users to
//       define their own custom templates for different blocks (meaning that
//       this should use text/template rather than using tabwriters manually.
func (ms ManifestStat) Format(w io.Writer) error {
	// Output history information.
	tw := tabwriter.NewWriter(w, 4, 2, 1, ' ', 0)
	fmt.Fprintf(tw, "LAYER\tCREATED\tCREATED BY\tSIZE\tCOMMENT\n")
	for _, histEntry := range ms.History {
		var (
			created   = strings.Replace(histEntry.Created.Format(igen.ISO8601), "\t", " ", -1)
			createdBy = strings.Replace(histEntry.CreatedBy, "\t", " ", -1)
			comment   = strings.Replace(histEntry.Comment, "\t", " ", -1)
			layerID   = "<none>"
			size      = "<none>"
		)

		if !histEntry.EmptyLayer {
			layerID = histEntry.Layer.Digest.String()
			size = units.HumanSize(float64(histEntry.Layer.Size))
		}

		// TODO: We need to truncate some of the fields.
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", layerID, created, createdBy, size, comment)
	}
	return tw.Flush()
}

// historyStat contains information about a single entry in the history of a
// manifest. This is essentially equivalent to a single record from
// docker-history(1).
type historyStat struct {
	// Layer is the descriptor referencing where the layer is stored. If it is
	// nil, then this entry is an empty_layer (and thus doesn't have a backing
	// diff layer).
	Layer *ispec.Descriptor `json:"layer"`

	// DiffID is an additional piece of information to Layer. It stores the
	// DiffID of the given layer corresponding to the history entry. If DiffID
	// is "", then this entry is an empty_layer.
	DiffID string `json:"diff_id"`

	// History is embedded in the stat information.
	ispec.History
}

// Stat computes the ManifestStat for a given manifest blob. The provided
// descriptor must refer to an OCI Manifest.
func Stat(ctx context.Context, engine casext.Engine, manifestDescriptor ispec.Descriptor) (ManifestStat, error) {
	var stat ManifestStat

	if manifestDescriptor.MediaType != ispec.MediaTypeImageManifest {
		return stat, errors.Errorf("stat: cannot stat a non-manifest descriptor: invalid media type '%s'", manifestDescriptor.MediaType)
	}

	// We have to get the actual manifest.
	manifestBlob, err := engine.FromDescriptor(ctx, manifestDescriptor)
	if err != nil {
		return stat, err
	}
	manifest, ok := manifestBlob.Data.(ispec.Manifest)
	if !ok {
		// Should _never_ be reached.
		return stat, errors.Errorf("[internal error] unknown manifest blob type: %s", manifestBlob.Descriptor.MediaType)
	}

	// Now get the config.
	configBlob, err := engine.FromDescriptor(ctx, manifest.Config)
	if err != nil {
		return stat, errors.Wrap(err, "stat")
	}
	config, ok := configBlob.Data.(ispec.Image)
	if !ok {
		// Should _never_ be reached.
		return stat, errors.Errorf("[internal error] unknown config blob type: %s", configBlob.Descriptor.MediaType)
	}

	// TODO: This should probably be moved into separate functions.

	// Generate the history of the image. Because the config.History entries
	// are in the same order as the manifest.Layer entries this is fairly
	// simple. However, we only increment the layer index if a layer was
	// actually generated by a history entry.
	layerIdx := 0
	for _, histEntry := range config.History {
		info := historyStat{
			History: histEntry,
			DiffID:  "",
			Layer:   nil,
		}

		// Only fill the other information and increment layerIdx if it's a
		// non-empty layer.
		if !histEntry.EmptyLayer {
			info.DiffID = config.RootFS.DiffIDs[layerIdx].String()
			info.Layer = &manifest.Layers[layerIdx]
			layerIdx++
		}

		stat.History = append(stat.History, info)
	}

	return stat, nil
}

// GenerateBundleManifest creates and writes an mtree of the rootfs in the given
// bundle path, using the supplied fsEval method
func GenerateBundleManifest(mtreeName string, bundlePath string, fsEval mtree.FsEval) error {
	mtreePath := filepath.Join(bundlePath, mtreeName+".mtree")
	fullRootfsPath := filepath.Join(bundlePath, layer.RootfsName)

	log.WithFields(log.Fields{
		"keywords": MtreeKeywords,
		"mtree":    mtreePath,
	}).Debugf("umoci: generating mtree manifest")

	log.Info("computing filesystem manifest ...")
	dh, err := mtree.Walk(fullRootfsPath, nil, MtreeKeywords, fsEval)
	if err != nil {
		return errors.Wrap(err, "generate mtree spec")
	}
	log.Info("... done")

	flags := os.O_CREATE | os.O_WRONLY | os.O_EXCL
	fh, err := os.OpenFile(mtreePath, flags, 0644)
	if err != nil {
		return errors.Wrap(err, "open mtree")
	}
	defer fh.Close()

	log.Debugf("umoci: saving mtree manifest")

	if _, err := dh.WriteTo(fh); err != nil {
		return errors.Wrap(err, "write mtree")
	}

	return nil
}

// ParseIdmapOptions sets up the mapping options for Meta, using
// the arguments specified on the command line
func ParseIdmapOptions(meta *Meta, ctx *cli.Context) error {
	// We need to set mappings if we're in rootless mode.
	meta.MapOptions.Rootless = ctx.Bool("rootless")
	if meta.MapOptions.Rootless {
		if !ctx.IsSet("uid-map") {
			if err := ctx.Set("uid-map", fmt.Sprintf("0:%d:1", os.Geteuid())); err != nil {
				// Should _never_ be reached.
				return errors.Wrap(err, "[internal error] failure auto-setting rootless --uid-map")
			}
		}
		if !ctx.IsSet("gid-map") {
			if err := ctx.Set("gid-map", fmt.Sprintf("0:%d:1", os.Getegid())); err != nil {
				// Should _never_ be reached.
				return errors.Wrap(err, "[internal error] failure auto-setting rootless --gid-map")
			}
		}
	}

	for _, uidmap := range ctx.StringSlice("uid-map") {
		idMap, err := idtools.ParseMapping(uidmap)
		if err != nil {
			return errors.Wrapf(err, "failure parsing --uid-map %s", uidmap)
		}
		meta.MapOptions.UIDMappings = append(meta.MapOptions.UIDMappings, idMap)
	}
	for _, gidmap := range ctx.StringSlice("gid-map") {
		idMap, err := idtools.ParseMapping(gidmap)
		if err != nil {
			return errors.Wrapf(err, "failure parsing --gid-map %s", gidmap)
		}
		meta.MapOptions.GIDMappings = append(meta.MapOptions.GIDMappings, idMap)
	}

	log.WithFields(log.Fields{
		"map.uid": meta.MapOptions.UIDMappings,
		"map.gid": meta.MapOptions.GIDMappings,
	}).Debugf("parsed mappings")

	return nil
}
