package c8y

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// MeasurementService does something
type MeasurementService service

// MeasurementCollectionOptions todo
type MeasurementCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	ValueFragmentType string `url:"valueFragmentType,omitempty"`

	ValueFragmentSeries string `url:"valueFragmentSeries,omitempty"`

	Revert bool `url:"revert,omitempty"`

	// Pagination options
	PaginationOptions
}

// MeasurementSeriesOptions todo
type MeasurementSeriesOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	AggregationType string `url:"aggregationType,omitempty"`

	Variables []string `url:"series,omitempty"`

	Revert bool `url:"revert,omitempty"`
}

// MeasurementCollection is the generic data structure which contains the response cumulocity when requesting a measurement collection
type MeasurementCollection struct {
	*BaseResponse

	Measurements []Measurement `json:"measurements"`

	Items []gjson.Result `json:"-"`
}

// Measurements represents multiple measurements
type Measurements struct {
	Measurements []MeasurementRepresentation `json:"measurements"`

	Items []gjson.Result `json:"-"`
}

// GetMeasurements return a measurement collection (multiple measurements)
func (s *MeasurementService) GetMeasurements(ctx context.Context, opt *MeasurementCollectionOptions) (*MeasurementCollection, *Response, error) {
	data := new(MeasurementCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "measurement/measurements",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteMeasurements removes a measurement collection
func (s *MeasurementService) DeleteMeasurements(ctx context.Context, opt *MeasurementCollectionOptions) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "measurement/measurements",
		Query:  opt,
	})
}

// MeasurementSeriesDefinition represents information about a single series
type MeasurementSeriesDefinition struct {
	Unit string `json:"unit"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// MeasurementSeriesValueGroup represents multiple values for multiple series for a single timestamp
type MeasurementSeriesValueGroup struct {
	Timestamp time.Time `json:"timestamp"`
	Values    []Number  `json:"values"`
}

// MeasurementSeriesAggregateValueGroup represents multiple aggregate values for multiple series for a single timestamp
type MeasurementSeriesAggregateValueGroup struct {
	Timestamp time.Time                   `json:"timestamp"`
	Values    []MeasurementAggregateValue `json:"values"`
}

// MeasurementSeriesGroup represents a group of series values (no aggregate values)
type MeasurementSeriesGroup struct {
	DeviceID  string                        `json:"deviceId"`
	Series    []MeasurementSeriesDefinition `json:"series"`
	Values    []MeasurementSeriesValueGroup `json:"values"`
	DateFrom  time.Time                     `json:"dateFrom"`
	DateTo    time.Time                     `json:"dateTo"`
	Truncated bool                          `json:"truncated"`
}

// MeasurementSeriesAggregateGroup represents a group of aggregate series
type MeasurementSeriesAggregateGroup struct {
	Series    []MeasurementSeriesDefinition          `json:"series"`
	Values    []MeasurementSeriesAggregateValueGroup `json:"values"`
	DateFrom  time.Time                              `json:"dateFrom"`
	DateTo    time.Time                              `json:"dateTo"`
	Truncated bool                                   `json:"truncated"`
}

// MeasurementAggregateValue represents the aggregate value of a single measurement.
type MeasurementAggregateValue struct {
	Min Number `json:"min"`
	Max Number `json:"max"`
}

// UnmarshalJSON converts the Cumulocity measurement Series response to a format which is easier to parse.
//
//	{
//	    "series": [ "c8y_Temperature.A", "c8y_Temperature.B" ],
//	    "unit": [ "degC", "degC" ],
//	    "truncated": true,
//	    "values": [
//	        { "timestamp": "2018-11-11T23:20:00.000+01:00", values: [0.0001, 0.1001] },
//	        { "timestamp": "2018-11-11T23:20:01.000+01:00", values: [0.1234, 2.2919] },
//	        { "timestamp": "2018-11-11T23:20:02.000+01:00", values: [0.8370, 4.8756] }
//	    ]
//	}
func (d *MeasurementSeriesGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			Logger.Infof("Could not unmarshal series definition. %s", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {
		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesValueGroup{
			Timestamp: timestamp,
			Values:    make([]Number, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			// Note: min and max values are the same when no aggregation is being used!
			// so technically we could get the value from either min or max.
			v := value.Get("max").String()

			seriesValues.Values[index] = *NewNumber(v)
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// UnmarshalJSON controls the conversion of json bytes to the MeasurementSeriesAggregateGroup struct
func (d *MeasurementSeriesAggregateGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			Logger.Infof("Could not unmarshal series definition. %s", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesAggregateValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {

		Logger.Infof("Key: %s", key)
		Logger.Infof("Values: %s", values)

		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesAggregateValueGroup{
			Timestamp: timestamp,
			Values:    make([]MeasurementAggregateValue, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			Logger.Infof("Current value: %s", value)
			v := &MeasurementAggregateValue{}
			json.Unmarshal([]byte(value.String()), &v)

			Logger.Infof("Full Value: %v", v)

			seriesValues.Values[index] = *v
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// GetMeasurementSeries returns the measurement series for a given source and variables
// The data is returned in a user friendly format to make it easier to use the data
func (s *MeasurementService) GetMeasurementSeries(ctx context.Context, opt *MeasurementSeriesOptions) (*MeasurementSeriesGroup, *Response, error) {
	u := "measurement/measurements/series"

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	Logger.Infof("query Parameters: %s", queryParams)

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(MeasurementSeriesGroup)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	// Add extra information to the results
	data.DeviceID = opt.Source

	return data, resp, nil
}

// GetMeasurement returns a single measurement
// Deprecated: Retrieving single measurements is no longer supported in Cumulocity IoT
// when using the time series feature. Use `GetMeasurements` instead
func (s *MeasurementService) GetMeasurement(ctx context.Context, ID string) (*Measurement, *Response, error) {
	data := new(Measurement)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "measurement/measurements/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removed a measurement by ID
// Deprecated: Deleting single measurements is no longer supported in Cumulocity IoT
// when using the time series feature. Use `DeleteMeasurements` instead
func (s *MeasurementService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "measurement/measurements/" + ID,
	})
}

// MarshalCSV converts the measurement series group to a csv output so it can be more easily parsed by other languages
// Example output
// timestamp,c8y_Temperature.A,c8y_Temperature.B
// 2018-11-23T00:45:39+01:00,60.699993,44.300003
// 2018-11-23T01:45:39+01:00,67.63333,47.199997
func (d *MeasurementSeriesGroup) MarshalCSV(delimiter string) ([]byte, error) {

	useDelimiter := delimiter
	if useDelimiter == "" {
		useDelimiter = ","
	}

	totalSeries := len(d.Series)

	// First column is the timestamp
	headers := make([]string, totalSeries+1)
	row := make([]string, totalSeries+1)

	headers[0] = "timestamp"

	var output string

	for i, header := range d.Series {
		headers[i+1] = fmt.Sprintf("%s.%s", header.Type, header.Name)
		output = strings.Join(headers, useDelimiter) + "\n"
	}

	for _, datapoint := range d.Values {
		row[0] = datapoint.Timestamp.Format(time.RFC3339)
		for i := 0; i < totalSeries; i++ {
			if datapoint.Values[i].IsNull() {
				row[i+1] = ""
			} else {
				row[i+1] = datapoint.Values[i].String()
			}
		}
		output += strings.Join(row, useDelimiter) + "\n"
	}

	return []byte(output), nil
}

// Create posts a new measurement to the platform
func (s *MeasurementService) Create(ctx context.Context, body MeasurementRepresentation) (*Measurement, *Response, error) {
	data := new(Measurement)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "measurement/measurements",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// CreateMeasurements posts multiple measurement to the platform
func (s *MeasurementService) CreateMeasurements(ctx context.Context, body *Measurements) (*Measurements, *Response, error) {
	data := new(Measurements)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "measurement/measurements",
		ContentType:  "application/vnd.com.nsn.cumulocity.measurementCollection+json",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}
