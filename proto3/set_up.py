from distutils.core import setup
from Cython.Build import cythonize

setup(
    ext_modules = cythonize("shaniu.py"),
    libraries=[
      "srpc_pb2",
      "srpc_pb2_grpc",
      "grpc",
      "google.protobuf",
      ]  # 指定第三方依赖项所在的库
)