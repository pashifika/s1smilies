/*
Version: 0.1
@author: Pashifika
*/
package libs

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type SmiliesDL struct {
	Stype  map[string]smiliesType // key is smiliesType ID
	DLlist map[string][]dlFile    // key is smiliesType ID
}
type smiliesType struct {
	Name    string
	DirPath string
}
type dlFile struct {
	Name     string
	FileName string
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

type jsonObjs [][]string

// Load smilies cache JavaScript to the memory
func LoadJStoMemory(url string, st_re, sa_re *regexp.Regexp, sf_index int) (*SmiliesDL, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the body to bytes
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the JavaScript file as Golang data frame
	var (
		match_st, match_sa [][]string
		ok                 bool
	)
	res := SmiliesDL{DLlist: map[string][]dlFile{}, Stype: map[string]smiliesType{}}
	for _, line := range strings.Split(string(buf), ";") {
		// Parse the smilies_type data
		if match_st = st_re.FindAllStringSubmatch(line, 2); len(match_st) >= 1 {
			if _, ok = res.Stype[match_st[0][1]]; !ok {
				res.Stype[match_st[0][1]] = smiliesType{
					Name:match_st[0][2],
					DirPath:match_st[0][3],
				}
			}
		}

		// Parse the smilies_array data
		if match_sa = sa_re.FindAllStringSubmatch(line, 2); len(match_sa) >= 1 {
			// Conv the smilies_array value to JSON object
			var smiliesObjs jsonObjs
			err = json.Unmarshal([]byte(strings.Replace(match_sa[0][2], `'`, `"`, -1)), &smiliesObjs)
			if err != nil {
				return nil, err
			}

			// Loop the JSON object and set it
			stID := match_sa[0][1]
			for _, sobj := range smiliesObjs {
				if _, ok = res.DLlist[stID]; !ok {
					res.DLlist[stID] = []dlFile{
						{Name: stID + "_" + sobj[sf_index], FileName: sobj[sf_index]},
					}
				} else {
					res.DLlist[stID] = append(res.DLlist[stID], dlFile{
						Name: stID + "_" + sobj[sf_index], FileName: sobj[sf_index],
					})
				}
			}
		}
	}

	return &res, nil
}
