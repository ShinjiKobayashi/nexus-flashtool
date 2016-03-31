package main

import "testing"


func Test_checkMd5(t *testing.T) {
	var correct = "11af8ee7c98ecaa2e405318007260e1e"
	if !checkMd5("shamu-lmy48m-factory-336efdae.tgz.tmp", correct) {
		t.Errorf("unmatch md5")
	}
}
