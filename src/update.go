package src

import (
	"errors"
	"fmt"
	"reflect"
)

func conditionalUpdateChanges() (err error) {

checkVersion:
	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settingsMap) == 0 {
		return
	}

	if settingsVersion, ok := settingsMap["version"].(string); ok {

		if settingsVersion > System.DBVersion {
			showInfo("Settings DB Version:" + settingsVersion)
			showInfo("System DB Version:" + System.DBVersion)
			err = errors.New(getErrMsg(1031))
			return
		}

		// Letzte Kompatible Version (1.4.4)
		if settingsVersion < System.Compatibility {
			err = errors.New(getErrMsg(1013))
			return
		}

		switch settingsVersion {

		case "1.4.4":
			// UUID Wert in xepg.json setzen
			err = setValueForUUID()
			if err != nil {
				return
			}

			// Neuer Filter (WebUI). Alte Filtereinstellungen werden konvertiert
			if oldFilter, ok := settingsMap["filter"].([]interface{}); ok {
				var newFilterMap = convertToNewFilter(oldFilter)
				settingsMap["filter"] = newFilterMap

				settingsMap["version"] = "2.0.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.0.0":

			if oldBuffer, ok := settingsMap["buffer"].(bool); ok {

				var newBuffer string
				switch oldBuffer {
				case true:
					// Native threadfin buffer was removed; FFmpeg is the supported replacement.
					newBuffer = "ffmpeg"
				case false:
					newBuffer = "-"
				}

				settingsMap["buffer"] = newBuffer

				settingsMap["version"] = "2.1.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.1.0":
			if normalizeLegacyBufferInSettingsMap(settingsMap) {
				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}
			}

			break
		}

	} else {
		// settings.json ist zu alt (älter als Version 1.4.4)
		err = errors.New(getErrMsg(1013))
	}

	return
}

// normalizeLegacyBufferMode maps removed native "threadfin" buffer to FFmpeg.
func normalizeLegacyBufferMode(buffer string) string {
	if buffer == "threadfin" {
		return "ffmpeg"
	}
	return buffer
}

func normalizeLegacyBufferInSettingsMap(settingsMap map[string]interface{}) (changed bool) {
	if b, ok := settingsMap["buffer"].(string); ok && b == "threadfin" {
		settingsMap["buffer"] = "ffmpeg"
		changed = true
	}

	files, ok := settingsMap["files"].(map[string]interface{})
	if !ok {
		return changed
	}

	for _, fileType := range []string{"m3u", "hdhr"} {
		playlists, ok := files[fileType].(map[string]interface{})
		if !ok {
			continue
		}
		for id, entry := range playlists {
			playlist, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			if b, ok := playlist["buffer"].(string); ok && b == "threadfin" {
				playlist["buffer"] = "ffmpeg"
				playlists[id] = playlist
				changed = true
			}
		}
		files[fileType] = playlists
	}

	return changed
}

func convertToNewFilter(oldFilter []interface{}) (newFilterMap map[int]interface{}) {

	newFilterMap = make(map[int]interface{})

	switch reflect.TypeOf(oldFilter).Kind() {

	case reflect.Slice:
		s := reflect.ValueOf(oldFilter)

		for i := 0; i < s.Len(); i++ {

			var newFilter FilterStruct
			newFilter.Active = true
			newFilter.Name = fmt.Sprintf("Custom filter %d", i+1)
			newFilter.Filter = s.Index(i).Interface().(string)
			newFilter.Type = "custom-filter"
			newFilter.CaseSensitive = false

			newFilterMap[i] = newFilter

		}

	}

	return
}

func setValueForUUID() (err error) {

	xepg, err := loadJSONFileToMap(System.File.XEPG)

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if uuidKey, ok := xepgChannel["_uuid.key"].(string); ok {

			if value, ok := xepgChannel[uuidKey].(string); ok {

				if len(value) > 0 {
					xepgChannel["_uuid.value"] = value
				}

			}

		}

	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)

	return
}
