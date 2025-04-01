package model

type Option func(*Config)

type OptionList struct {
	Options []Option `yaml:""`
}
