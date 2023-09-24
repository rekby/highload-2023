import os
import tempfile
import shutil
import os.path

import pytest


def test_create_dir(folder):
    path = os.path.join(folder, "tmp")
    os.mkdir(path)


def test_create_file(folder):
    path = os.path.join(folder, "tmp")
    f = open(path, "wb")
    f.close()


@pytest.fixture()
def folder():
    dir = tempfile.mkdtemp()
    yield dir
    shutil.rmtree(dir)
