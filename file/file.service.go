package file

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"gopkg.in/mgo.v2/bson"

	"fileServer/config"
	"fileServer/db/mongo"
	"fileServer/keys"
)

// WalkDir 获取指定目录及所有子目录下的所有文件，可以匹配后缀过滤。
func WalkDir(dirPth, suffix string) ([]string, error) {
	files := make([]string, 0, 30)
	suffix = strings.ToUpper(suffix)
	err := filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		// 忽略目录
		if fi.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

// ReadFile 读取文件内容
func ReadFile(path string) (string, error) {
	fileHandle, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fileHandle.Close()
	fileBytes, err := ioutil.ReadAll(fileHandle)
	return string(fileBytes), err
}

// IsExist 文件是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// GetUniqueName 获取一个唯一的文件名
func GetUniqueName() string {
	id := bson.NewObjectId()
	newName := id.Hex()
	return newName
}

// Put 存储文件到数据库
func Put(name string, f interface{}, meta interface{}) (string, error) {
	if f == nil {
		log.Error().Str("func", "file.Put").Msg("文件指针为空.")
		return "", errors.New(keys.ErrorFileInfo)
	}
	mongoSession := mongo.Session.Clone()
	defer mongoSession.Close()

	mongoFile, err := mongoSession.DB(config.App.Mongo.Database).GridFS(mongo.Files).Create(name)
	if err != nil {
		log.Error().Err(err).Str("func", "file.Put").Msg("Fail to create a file on mongo.")
		return "", err
	}

	defer mongoFile.Close()
	var fileID string

	fid := mongoFile.Id()
	switch v := fid.(type) {
	case string:
		fileID = v
	case bson.ObjectId:
		fileID = v.Hex()
	}
	if name == "" {
		mongoFile.SetName(fileID)
	}
	mongoFile.SetMeta(meta)
	switch data := f.(type) {
	case []byte:
		_, err = mongoFile.Write(data)
	case io.Reader:
		_, err = io.Copy(mongoFile, data)
	default:
	}
	if err != nil {
		log.Error().Err(err).Str("func", "file.Put").Msg("Fail to write file on mongo.")
		return "", err
	}
	return fileID, err
}

// Info 从数据库读取文件基础信息
func Info(name string) (*FileInfo, error) {
	if name == "" {
		return nil, errors.New(keys.ErrorParam)
	}
	mongoSession := mongo.Session.Clone()
	defer mongoSession.Close()

	fileInfo := &FileInfo{}
	err := mongoSession.DB(config.App.Mongo.Database).GridFS(mongo.Files).Find(bson.M{
		"filename": name,
	}).Select(bson.M{"filename": true, "metadata": true}).One(&fileInfo)

	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// Get 从数据库读取文件
func Get(name, dir string) error {
	if name == "" {
		return errors.New(keys.ErrorParam)
	}
	mongoSession := mongo.Session.Clone()
	defer mongoSession.Close()

	file, err := mongoSession.DB(config.App.Mongo.Database).GridFS(mongo.Files).Open(name)
	if err != nil {
		log.Error().Caller().Err(err).Str("func", "file.Get").Msgf("Fail to open from GridFS. Name=%s", name)
		return err
	}
	defer file.Close()

	fullname := dir + name
	cachePath := path.Dir(fullname)
	if !IsExist(cachePath) {
		err := os.MkdirAll(cachePath, os.ModePerm)
		if err != nil {
			log.Panic().Caller().Err(err).Msgf("Fail to create the CachePath: %s", cachePath)
		}
	}

	fw, err := os.Create(fullname)
	if err != nil {
		log.Error().Caller().Err(err).Msgf("Fail to create file. fullname=%s", fullname)
		return err
	}
	defer fw.Close()
	_, err = io.Copy(fw, file)
	return err
}

// ImageThumbnail 生成图片缩略图
func ImageThumbnail(src string, w, h int) (string, error) {
	filename := fmt.Sprintf("%s_%d_%d", src, w, h)

	if IsExist(filename) {
		return filename, nil
	}

	file, err := os.Open(src)
	if err != nil {
		return src, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return src, err
	}

	thumb := resize.Thumbnail(uint(w), uint(h), img, resize.Lanczos3)
	out, err := os.Create(filename)
	if err != nil {
		return src, err
	}
	defer out.Close()

	// Write new image to file.
	err = jpeg.Encode(out, thumb, nil)

	if err != nil {
		return src, err
	}

	return filename, nil
}
