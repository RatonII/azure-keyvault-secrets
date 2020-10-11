package main

type Functions []Function

type Function struct {
	Name 			 	string	`yaml:"name"`
	SecretKeyName 		string	`yaml:"secretkeyname"`
}


