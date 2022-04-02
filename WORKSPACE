load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "d6b2513456fe2229811da7eb67a444be7785f5323c6708b38d851d2b51e54d83",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.30.0/rules_go-v0.30.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.30.0/rules_go-v0.30.0.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "1.18")

http_archive(
    name = "bazel_gazelle",
    sha256 = "5982e5463f171da99e3bdaeff8c0f48283a7a5f396ec5282910b9e8a49c0dd7e",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.25.0/bazel-gazelle-v0.25.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.25.0/bazel-gazelle-v0.25.0.tar.gz",
    ],
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

go_repository(
    name = "org_golang_x_xerrors",
    importpath = "golang.org/x/xerrors",
    sum = "h1:go1bK/D/BFZV2I8cIQd1NKEZ+0owSTG1fDTci4IqFcE=",
    version = "v0.0.0-20200804184101-5ec99f83aff1",
)

go_repository(
    name = "org_bazil_fuse",
    importpath = "bazil.org/fuse",
    sum = "h1:UrYe9YkT4Wpm6D+zByEyCJQzDqTPXqTDUI7bZ41i9VE=",
    version = "v0.0.0-20200524192727-fb710f7dfd05",
)

go_repository(
    name = "com_github_dchest_blake2b",
    importpath = "github.com/dchest/blake2b",
    sum = "h1:KK9LimVmE0MjRl9095XJmKqZ+iLxWATvlcpVFRtaw6s=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_dustin_go_humanize",
    importpath = "github.com/dustin/go-humanize",
    sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    importpath = "github.com/hashicorp/golang-lru",
    sum = "h1:YDjusn29QI/Das2iO9M0BHnIbxPeyuCHsjMW+lJfyTc=",
    version = "v0.5.4",
)

go_repository(
    name = "com_github_jinzhu_now",
    importpath = "github.com/jinzhu/now",
    sum = "h1:/o9tlHleP7gOFmsnYNz3RGnqzefHA47wQpKrrdTIwXQ=",
    version = "v1.1.5",
)

go_repository(
    name = "com_github_robfig_cron",
    importpath = "github.com/robfig/cron",
    sum = "h1:ZjScXvvxeQ63Dbyxy76Fj3AT3Ut0aKsyd2/tl3DTMuQ=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_workiva_go_datastructures",
    importpath = "github.com/Workiva/go-datastructures",
    sum = "h1:J6Y/52yX10Xc5JjXmGtWoSSxs3mZnGSaq37xZZh7Yig=",
    version = "v1.0.53",
)

go_repository(
    name = "com_github_boltdb_bolt",
    importpath = "github.com/boltdb/bolt",
    sum = "h1:JQmyP4ZBrce+ZQu0dY660FMfatumYDLun9hBCUVIkF4=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:ZDRjVQ15GmhC3fiQ8ni8+OwkZQO4DARzQgrnXU1Liz8=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_dvyukov_go_fuzz",
    importpath = "github.com/dvyukov/go-fuzz",
    sum = "h1:NgO45/5mBLRVfiXerEFzH6ikcZ7DNRPS639xFg3ENzU=",
    version = "v0.0.0-20200318091601-be3528f3a813",
)

go_repository(
    name = "com_github_elazarl_go_bindata_assetfs",
    importpath = "github.com/elazarl/go-bindata-assetfs",
    sum = "h1:G/bYguwHIzWq9ZoyUQqrjTmJbbYn3j3CKKpKinvZLFk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_julusian_godocdown",
    importpath = "github.com/Julusian/godocdown",
    sum = "h1:n3F+mWm+b4D7uNbx1syN/uQTVDwt2sWfk23Mhzwzec4=",
    version = "v0.0.0-20170816220326-6d19f8ff2df8",
)

go_repository(
    name = "com_github_philhofer_fwd",
    importpath = "github.com/philhofer/fwd",
    sum = "h1:GdGcTjf5RNAxwS4QLsiMzJYj5KEvPJD3Abr261yRQXQ=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_pmezard_go_difflib",
    importpath = "github.com/pmezard/go-difflib",
    sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_robertkrimen_godocdown",
    importpath = "github.com/robertkrimen/godocdown",
    sum = "h1:jMxcLa+VjJKhpCwbLUXAD15wJ+hhvXMLujCl3MkXpfM=",
    version = "v0.0.0-20130622164427-0bfa04905481",
)

go_repository(
    name = "com_github_sabhiram_go_gitignore",
    importpath = "github.com/sabhiram/go-gitignore",
    sum = "h1:OkMGxebDjyw0ULyrTYWeN0UNCCkmCWfjPnIA2W6oviI=",
    version = "v0.0.0-20210923224102-525f6e181f06",
)

go_repository(
    name = "com_github_stephens2424_writerset",
    importpath = "github.com/stephens2424/writerset",
    sum = "h1:znRLgU6g8RS5euYRcy004XeE4W+Tu44kALzy7ghPif8=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_stretchr_objx",
    importpath = "github.com/stretchr/objx",
    sum = "h1:4G4v2dO3VZwixGIRoQ5Lfboy6nUhCyYzaqnIAPPhYs4=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    sum = "h1:nwc3DEeHmmLAfoZucVR881uASk0Mfjw8xYJ99tb5CcY=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_tinylib_msgp",
    importpath = "github.com/tinylib/msgp",
    sum = "h1:2gXmtWueD2HefZHQe1QOy9HVzmFrLOVvsXwXBQ0ayy0=",
    version = "v1.1.5",
)

go_repository(
    name = "com_github_ttacon_chalk",
    importpath = "github.com/ttacon/chalk",
    sum = "h1:OXcKh35JaYsGMRzpvFkLv/MEyPuL49CThT1pZ8aSml4=",
    version = "v0.0.0-20160626202418-22c06c80ed31",
)

go_repository(
    name = "com_github_tv42_httpunix",
    importpath = "github.com/tv42/httpunix",
    sum = "h1:u6SKchux2yDvFQnDHS3lPnIRmfVJ5Sxy3ao2SIdysLQ=",
    version = "v0.0.0-20191220191345-2ba4b9c3382c",
)

go_repository(
    name = "com_github_yuin_goldmark",
    importpath = "github.com/yuin/goldmark",
    sum = "h1:ruQGxdhGHe7FWOJPT0mKs5+pD2Xs1Bm/kdGlHO04FmM=",
    version = "v1.2.1",
)

go_repository(
    name = "in_gopkg_check_v1",
    importpath = "gopkg.in/check.v1",
    sum = "h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=",
    version = "v0.0.0-20161208181325-20d25e280405",
)

go_repository(
    name = "in_gopkg_yaml_v3",
    importpath = "gopkg.in/yaml.v3",
    sum = "h1:dUUwHk2QECo/6vqA44rthZ8ie2QXMNeKRTHCNY2nXvo=",
    version = "v3.0.0-20200313102051-9f266ea9e77c",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:psW17arqaxU48Z5kZ0CQnkZWQJsqcURM6tKiBApRjXI=",
    version = "v0.0.0-20200622213623-75b288015ac9",
)

go_repository(
    name = "org_golang_x_mod",
    importpath = "golang.org/x/mod",
    sum = "h1:RM4zey1++hCTbCVQfnWeKs9/IEsaBLA8vTkd0WVtmH4=",
    version = "v0.3.0",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:IX6qOQeG5uLjB/hjjwjedwfjND0hgjPMMyO1RoIXQNI=",
    version = "v0.0.0-20201021035429-f5854403a974",
)

go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    sum = "h1:SQFwaSi55rU7vdNs9Yr0Z324VNlrF+0wMqRXT4St8ck=",
    version = "v0.0.0-20201020160332-67f06af15bc9",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    sum = "h1:+Nyd8tzPX9R7BWHguqsrbFdRx3WQ/1ib8I44HXV5yTA=",
    version = "v0.0.0-20200930185726-fdedc70b468f",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:cokOdA+Jmi5PJGXLlLllQSgYigAEfHXJAERHVMaCc2k=",
    version = "v0.3.3",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:sEvmEcJVKBNUvgCUClbUQeHOAa9U0I2Ce1BooMvVCY4=",
    version = "v0.0.0-20201022035929-9cf592e881e9",
)

gazelle_dependencies()
