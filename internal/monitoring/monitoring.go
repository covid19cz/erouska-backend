package monitoring

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"time"

	googlemonitoring "cloud.google.com/go/monitoring/apiv3/v2"
	googlemonitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

//MonitoringClient -_-
var MonitoringClient *googlemonitoring.MetricClient

func init() {
	ctx := context.Background()

	var err error
	MonitoringClient, err = googlemonitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("googlemonitoring.NewMetricClient: %v", err)
	}
}

//Reader Interface for monitoring client.
type Reader interface {
	ReadSummarized(ctx context.Context, projectID string, filter string, from time.Time, until time.Time, sumWindow int64) ([]int32, error)
}

//Client Real Monitoring client.
type Client struct{}

//ReadSummarized Reads summarized metrics. Specify point in history and window of aggregation.
func (c Client) ReadSummarized(ctx context.Context, projectID string, filter string, from time.Time, until time.Time, sumWindow int64) ([]int32, error) {
	logger := logging.FromContext(ctx)

	req := &googlemonitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%v", projectID),
		Filter: filter,
		Interval: &googlemonitoringpb.TimeInterval{
			StartTime: timestamppb.New(from),
			EndTime:   timestamppb.New(until),
		},
		Aggregation: &googlemonitoringpb.Aggregation{
			AlignmentPeriod: &duration.Duration{
				Seconds: sumWindow,
			},
			CrossSeriesReducer: googlemonitoringpb.Aggregation_REDUCE_SUM,
			PerSeriesAligner:   googlemonitoringpb.Aggregation_ALIGN_SUM,
		},
		View: googlemonitoringpb.ListTimeSeriesRequest_FULL,
	}

	it := MonitoringClient.ListTimeSeries(ctx, req)

	var items []int32

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not read time series value: %v", err)
		}

		for _, point := range resp.Points {
			logger.Debugf("Found value: [%v-%v] = %v\n",
				point.Interval.StartTime.Seconds,
				point.Interval.EndTime.Seconds,
				point.GetValue().GetInt64Value())

			items = append(items, int32(point.GetValue().GetInt64Value()))
		}

		resp.Reset()
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("Could not gather any values - empty result")
	}

	logger.Debugf("Gathered metrics values: %+v", items)

	return items, nil
}

//MockClient NOOP Monitoring client.
type MockClient struct{}

//ReadSummarized Does nothing.
func (m MockClient) ReadSummarized(ctx context.Context, projectID string, filter string, from time.Time, until time.Time, sumWindow int64) ([]int32, error) {
	return []int32{42}, nil
}
