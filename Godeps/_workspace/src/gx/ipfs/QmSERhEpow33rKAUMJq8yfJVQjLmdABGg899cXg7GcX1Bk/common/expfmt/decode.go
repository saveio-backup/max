// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expfmt

import (
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"

	dto "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYkNhwAviNzN974MB3koxuBRhtbvCotnuQcugrPF96BPp/client_model/go"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSERhEpow33rKAUMJq8yfJVQjLmdABGg899cXg7GcX1Bk/common/model"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUsh7scAkKe2erRhfGJpvxokArYL46ye2EJvFEp6XSbLr/golang_protobuf_extensions/pbutil"
)

// Decoder types decode an input stream into metric families.
type Decoder interface {
	Decode(*dto.MetricFamily) error
}

type DecodeOptions struct {
	// Timestamp is added to each value from the stream that has no explicit timestamp set.
	Timestamp model.Time
}

// ResponseFormat extracts the correct format from a HTTP response header.
// If no matching format can be found FormatUnknown is returned.
func ResponseFormat(h http.Header) Format {
	ct := h.Get(hdrContentType)

	mediatype, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return FmtUnknown
	}

	const (
		textType = "text/plain"
		jsonType = "application/json"
	)

	switch mediatype {
	case ProtoType:
		if p, ok := params["proto"]; ok && p != ProtoProtocol {
			return FmtUnknown
		}
		if e, ok := params["encoding"]; ok && e != "delimited" {
			return FmtUnknown
		}
		return FmtProtoDelim

	case textType:
		if v, ok := params["version"]; ok && v != TextVersion {
			return FmtUnknown
		}
		return FmtText

	case jsonType:
		var prometheusAPIVersion string

		if params["schema"] == "prometheus/telemetry" && params["version"] != "" {
			prometheusAPIVersion = params["version"]
		} else {
			prometheusAPIVersion = h.Get("X-Prometheus-API-Version")
		}

		switch prometheusAPIVersion {
		case "0.0.2", "":
			return fmtJSON2
		default:
			return FmtUnknown
		}
	}

	return FmtUnknown
}

// NewDecoder returns a new decoder based on the given input format.
// If the input format does not imply otherwise, a text format decoder is returned.
func NewDecoder(r io.Reader, format Format) Decoder {
	switch format {
	case FmtProtoDelim:
		return &protoDecoder{r: r}
	case fmtJSON2:
		return newJSON2Decoder(r)
	}
	return &textDecoder{r: r}
}

// protoDecoder implements the Decoder interface for protocol buffers.
type protoDecoder struct {
	r io.Reader
}

// Decode implements the Decoder interface.
func (d *protoDecoder) Decode(v *dto.MetricFamily) error {
	_, err := pbutil.ReadDelimited(d.r, v)
	return err
}

// textDecoder implements the Decoder interface for the text protcol.
type textDecoder struct {
	r    io.Reader
	p    TextParser
	fams []*dto.MetricFamily
}

// Decode implements the Decoder interface.
func (d *textDecoder) Decode(v *dto.MetricFamily) error {
	// TODO(fabxc): Wrap this as a line reader to make streaming safer.
	if len(d.fams) == 0 {
		// No cached metric families, read everything and parse metrics.
		fams, err := d.p.TextToMetricFamilies(d.r)
		if err != nil {
			return err
		}
		if len(fams) == 0 {
			return io.EOF
		}
		d.fams = make([]*dto.MetricFamily, 0, len(fams))
		for _, f := range fams {
			d.fams = append(d.fams, f)
		}
	}

	*v = *d.fams[0]
	d.fams = d.fams[1:]

	return nil
}

type SampleDecoder struct {
	Dec  Decoder
	Opts *DecodeOptions

	f dto.MetricFamily
}

func (sd *SampleDecoder) Decode(s *model.Vector) error {
	if err := sd.Dec.Decode(&sd.f); err != nil {
		return err
	}
	*s = extractSamples(&sd.f, sd.Opts)
	return nil
}

// Extract samples builds a slice of samples from the provided metric families.
func ExtractSamples(o *DecodeOptions, fams ...*dto.MetricFamily) model.Vector {
	var all model.Vector
	for _, f := range fams {
		all = append(all, extractSamples(f, o)...)
	}
	return all
}

func extractSamples(f *dto.MetricFamily, o *DecodeOptions) model.Vector {
	switch f.GetType() {
	case dto.MetricType_COUNTER:
		return extractCounter(o, f)
	case dto.MetricType_GAUGE:
		return extractGauge(o, f)
	case dto.MetricType_SUMMARY:
		return extractSummary(o, f)
	case dto.MetricType_UNTYPED:
		return extractUntyped(o, f)
	case dto.MetricType_HISTOGRAM:
		return extractHistogram(o, f)
	}
	panic("expfmt.extractSamples: unknown metric family type")
}

func extractCounter(o *DecodeOptions, f *dto.MetricFamily) model.Vector {
	samples := make(model.Vector, 0, len(f.Metric))

	for _, m := range f.Metric {
		if m.Counter == nil {
			continue
		}

		lset := make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName())

		smpl := &model.Sample{
			Metric: model.Metric(lset),
			Value:  model.SampleValue(m.Counter.GetValue()),
		}

		if m.TimestampMs != nil {
			smpl.Timestamp = model.TimeFromUnixNano(*m.TimestampMs * 1000000)
		} else {
			smpl.Timestamp = o.Timestamp
		}

		samples = append(samples, smpl)
	}

	return samples
}

func extractGauge(o *DecodeOptions, f *dto.MetricFamily) model.Vector {
	samples := make(model.Vector, 0, len(f.Metric))

	for _, m := range f.Metric {
		if m.Gauge == nil {
			continue
		}

		lset := make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName())

		smpl := &model.Sample{
			Metric: model.Metric(lset),
			Value:  model.SampleValue(m.Gauge.GetValue()),
		}

		if m.TimestampMs != nil {
			smpl.Timestamp = model.TimeFromUnixNano(*m.TimestampMs * 1000000)
		} else {
			smpl.Timestamp = o.Timestamp
		}

		samples = append(samples, smpl)
	}

	return samples
}

func extractUntyped(o *DecodeOptions, f *dto.MetricFamily) model.Vector {
	samples := make(model.Vector, 0, len(f.Metric))

	for _, m := range f.Metric {
		if m.Untyped == nil {
			continue
		}

		lset := make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName())

		smpl := &model.Sample{
			Metric: model.Metric(lset),
			Value:  model.SampleValue(m.Untyped.GetValue()),
		}

		if m.TimestampMs != nil {
			smpl.Timestamp = model.TimeFromUnixNano(*m.TimestampMs * 1000000)
		} else {
			smpl.Timestamp = o.Timestamp
		}

		samples = append(samples, smpl)
	}

	return samples
}

func extractSummary(o *DecodeOptions, f *dto.MetricFamily) model.Vector {
	samples := make(model.Vector, 0, len(f.Metric))

	for _, m := range f.Metric {
		if m.Summary == nil {
			continue
		}

		timestamp := o.Timestamp
		if m.TimestampMs != nil {
			timestamp = model.TimeFromUnixNano(*m.TimestampMs * 1000000)
		}

		for _, q := range m.Summary.Quantile {
			lset := make(model.LabelSet, len(m.Label)+2)
			for _, p := range m.Label {
				lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
			}
			// BUG(matt): Update other names to "quantile".
			lset[model.LabelName(model.QuantileLabel)] = model.LabelValue(fmt.Sprint(q.GetQuantile()))
			lset[model.MetricNameLabel] = model.LabelValue(f.GetName())

			samples = append(samples, &model.Sample{
				Metric:    model.Metric(lset),
				Value:     model.SampleValue(q.GetValue()),
				Timestamp: timestamp,
			})
		}

		lset := make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_sum")

		samples = append(samples, &model.Sample{
			Metric:    model.Metric(lset),
			Value:     model.SampleValue(m.Summary.GetSampleSum()),
			Timestamp: timestamp,
		})

		lset = make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_count")

		samples = append(samples, &model.Sample{
			Metric:    model.Metric(lset),
			Value:     model.SampleValue(m.Summary.GetSampleCount()),
			Timestamp: timestamp,
		})
	}

	return samples
}

func extractHistogram(o *DecodeOptions, f *dto.MetricFamily) model.Vector {
	samples := make(model.Vector, 0, len(f.Metric))

	for _, m := range f.Metric {
		if m.Histogram == nil {
			continue
		}

		timestamp := o.Timestamp
		if m.TimestampMs != nil {
			timestamp = model.TimeFromUnixNano(*m.TimestampMs * 1000000)
		}

		infSeen := false

		for _, q := range m.Histogram.Bucket {
			lset := make(model.LabelSet, len(m.Label)+2)
			for _, p := range m.Label {
				lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
			}
			lset[model.LabelName(model.BucketLabel)] = model.LabelValue(fmt.Sprint(q.GetUpperBound()))
			lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_bucket")

			if math.IsInf(q.GetUpperBound(), +1) {
				infSeen = true
			}

			samples = append(samples, &model.Sample{
				Metric:    model.Metric(lset),
				Value:     model.SampleValue(q.GetCumulativeCount()),
				Timestamp: timestamp,
			})
		}

		lset := make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_sum")

		samples = append(samples, &model.Sample{
			Metric:    model.Metric(lset),
			Value:     model.SampleValue(m.Histogram.GetSampleSum()),
			Timestamp: timestamp,
		})

		lset = make(model.LabelSet, len(m.Label)+1)
		for _, p := range m.Label {
			lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
		}
		lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_count")

		count := &model.Sample{
			Metric:    model.Metric(lset),
			Value:     model.SampleValue(m.Histogram.GetSampleCount()),
			Timestamp: timestamp,
		}
		samples = append(samples, count)

		if !infSeen {
			// Append an infinity bucket sample.
			lset := make(model.LabelSet, len(m.Label)+2)
			for _, p := range m.Label {
				lset[model.LabelName(p.GetName())] = model.LabelValue(p.GetValue())
			}
			lset[model.LabelName(model.BucketLabel)] = model.LabelValue("+Inf")
			lset[model.MetricNameLabel] = model.LabelValue(f.GetName() + "_bucket")

			samples = append(samples, &model.Sample{
				Metric:    model.Metric(lset),
				Value:     count.Value,
				Timestamp: timestamp,
			})
		}
	}

	return samples
}
