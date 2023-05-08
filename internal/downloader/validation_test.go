package downloader

import "testing"

func TestValidateLink(t *testing.T) {
	type args struct {
		link string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should_return_err_on_cyrillic_text",
			args: args{
				link: "Статус",
			},
			wantErr: true,
		},
		{
			name: "should_return_err_on_video_id",
			args: args{
				link: "7UxNoFjmhBA",
			},
			wantErr: true,
		},
		{
			name: "should_not_return_err_when_correct_youtube_url",
			args: args{
				link: "https://www.youtube.com/watch?v=7UxNoFjmhBA",
			},
			wantErr: false,
		},
		{
			name: "should_not_return_err_when_correct_youtube_url_without_schema",
			args: args{
				link: "youtube.com/watch?v=7UxNoFjmhBA",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateLink(tt.args.link); (err != nil) != tt.wantErr {
				t.Errorf("ValidateLink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
