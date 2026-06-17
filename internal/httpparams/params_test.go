package httpparams

import "testing"

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "valid", raw: "42", want: 42},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
		{name: "text", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PositiveInt(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("PositiveInt(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("PositiveInt(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestOptionalIntInRange(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "empty default", raw: "", want: 10},
		{name: "valid min", raw: "1", want: 1},
		{name: "valid max", raw: "20", want: 20},
		{name: "below", raw: "0", wantErr: true},
		{name: "above", raw: "21", wantErr: true},
		{name: "text", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptionalIntInRange(tt.raw, 10, 1, 20)
			if (err != nil) != tt.wantErr {
				t.Fatalf("OptionalIntInRange(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("OptionalIntInRange(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestOptionalFloatMin(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    float64
		wantErr bool
	}{
		{name: "empty default", raw: "", want: 0},
		{name: "valid", raw: "1.5", want: 1.5},
		{name: "equal min", raw: "0", want: 0},
		{name: "below", raw: "-0.1", wantErr: true},
		{name: "text", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptionalFloatMin(tt.raw, 0)
			if (err != nil) != tt.wantErr {
				t.Fatalf("OptionalFloatMin(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("OptionalFloatMin(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestPageOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		want     int
	}{
		{page: 1, pageSize: 100, want: 0},
		{page: 2, pageSize: 100, want: 100},
		{page: 0, pageSize: 100, want: 0},
	}

	for _, tt := range tests {
		if got := PageOffset(tt.page, tt.pageSize); got != tt.want {
			t.Fatalf("PageOffset(%d, %d) = %d, want %d", tt.page, tt.pageSize, got, tt.want)
		}
	}
}
