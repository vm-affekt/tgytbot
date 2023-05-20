package downloader

import (
	"context"
	"testing"

	"github.com/vm-affekt/tgytbot/internal/logging"
	"go.uber.org/zap"
)

func TestService_transformLink(t *testing.T) {
	ctx := context.Background()
	logging.SetLogger(zap.NewExample())
	type args struct {
		link string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should_return_same_url",
			args: args{
				link: "https://youtu.be/GQtVIUdr4sk",
			},
			want: "https://youtu.be/GQtVIUdr4sk",
		},
		{
			name: "should_extract_id_from_live_path",
			args: args{
				link: "https://www.youtube.com/live/SL6b1Shryww?feature=share",
			},
			want: "SL6b1Shryww",
		},
		{
			name: "should_not_return_idx_out_of_range_on_invalid_path",
			args: args{
				link: "https://www.youtube.com/live/?feature=share",
			},
			want: "https://www.youtube.com/live/?feature=share",
		},
		{
			name: "should_not_return_idx_out_of_range_on_invalid_path_without_query",
			args: args{
				link: "https://www.youtube.com/live/",
			},
			want: "https://www.youtube.com/live/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := new(Service)
			got := s.transformLink(ctx, tt.args.link)
			if got != tt.want {
				t.Errorf("Service.transformLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
