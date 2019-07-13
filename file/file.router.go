package file

import (
	"fileServer/middlewares"

	"github.com/kataras/iris"
)

// Route file模块路由
func Route(route *iris.Application) {

	route.Get("/file/{path:path}", File)
	route.Post("/api/file", middlewares.Auth, Limit, UploadFile)
	route.Delete("/api/file/{path:path}", middlewares.Auth, DelFile)

	route.Get("/image/{path:path}", Image)
	route.Post("/api/image", middlewares.Auth, Limit, UploadImage)
	route.Delete("/api/image/{path:path}", middlewares.Auth, DelFile)
}
