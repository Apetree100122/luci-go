#!/usr/bin/env python3
# Copyright 2016 The LUCI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Manages web/ resource checkout and building.
"""

import argparse
import itertools
import logging
import os
import pipes
import shutil
import subprocess
import sys

from distutils.spawn import find_executable

LOGGER = logging.getLogger('web.py')

# The root of the "luci-go" checkout, relative to the current "build.py" file.
_LUCI_GO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# The default build output path.
_WEB_BUILD_PATH = os.path.join(_LUCI_GO_ROOT, 'web')

class Toolchain(object):
  """Web toolchain management."""

  def __init__(self, web_dir, node_exe, npm_exe):
    self._web_dir = web_dir
    self._node_exe = node_exe
    self._npm_exe = npm_exe

  @classmethod
  def initialize(cls, source_root, force=False):
    web_dir = os.path.join(source_root, 'web')

    # Node and NPM must already be installed.
    node_js = [
        find_executable('node'),
        find_executable('npm'),
    ]
    if not all(node_js):
      print("""\
  Build requires a "node" and "npm" executables to be installed on your local
  system. Please install Node.js and NPM. Installation instructions can be found
  at:
      https://docs.npmjs.com/getting-started/installing-node
  """)
      raise Exception('Unable to locate Node.js installation.')
    tc = cls(web_dir, *node_js)

    # Install NPM deps from "package.json".
    def install_npm_deps():
      tc.npm('install', cwd=web_dir)
      tc.npm('prune', cwd=web_dir)
    cls._call_if_outdated(
        install_npm_deps,
        os.path.join(web_dir, '.npm.installed'),
        os.path.join(web_dir, 'package.json'),
        [os.path.join(web_dir, 'node_modules')],
        force)

    # Install Bower deps from "bower.json".
    def install_bower_deps():
      tc.bower('install', cwd=web_dir)
    cls._call_if_outdated(
        install_bower_deps,
        os.path.join(web_dir, '.bower.installed'),
        os.path.join(web_dir, 'bower.json'),
        [os.path.join(web_dir, 'inc', 'bower_components')],
        force)

    return tc

  @staticmethod
  def _call_if_outdated(fn, manifest_path, defs_path, clean_paths, force):
    """Will call "fn" if the file at "install_path" doesn't match "spec".

    If "fn" completes without raising an exception, the "spec" file will be
    copied to the "installed" path, making subsequent runs of this function a
    no-op until the spec file changes..

    Args:
      fn (callable): The function to call if they don't match.
      manifest_path (str): The path to the installed state file.
      defs_path (str): The path to the source spec file.
      clean_paths (list): Path of destination files and directories to clean on
          reprovision.
      force (bool): If true, call the function regardless.
    """
    with open(defs_path, 'r') as fd:
      spec_data = fd.read()

    if not force and os.path.isfile(manifest_path):
      with open(manifest_path, 'r') as fd:
        current = fd.read()
      if spec_data == current:
        return

    # Clean all paths.
    for path in itertools.chain(clean_paths, (manifest_path,)):
      if os.path.isdir(path):
        LOGGER.info('Purging directory on reprovision: %r', path)
        shutil.rmtree(path)
      elif os.path.isfile(path):
        LOGGER.info('Purging file on reprovision: %r', path)
        os.remove(path)

    # Either forcing or out of date.
    fn()

    # Update our installed file to match our spec data.
    with open(manifest_path, 'w') as fd:
      fd.write(spec_data)

  @property
  def web_dir(self):
    return self._web_dir

  @property
  def apps_dir(self):
    return os.path.join(self._web_dir, 'apps')

  def _call(self, *args, **kwargs):
    LOGGER.debug('Running command (cwd=%s): %s',
        kwargs.get('cwd', os.getcwd()),
        pipes.quote(' '.join(args)))

    kwargs['stderr'] = subprocess.STDOUT
    try:
      subprocess.check_call(args, **kwargs)
    except subprocess.CalledProcessError as e:
      LOGGER.warning('Non-zero return code (%d) from command.',
          e.returncode, exc_info=LOGGER.isEnabledFor(logging.DEBUG))
      sys.exit(e.returncode)

  def node(self, *args, **kwargs):
    self._call(self._node_exe, *args, **kwargs)

  def npm(self, *args, **kwargs):
    self._call(self._npm_exe, *args, **kwargs)

  def bower(self, *args, **kwargs):
    exe = os.path.join(self.web_dir, 'node_modules', 'bower', 'bin', 'bower')
    return self.node(exe, *args, **kwargs)

  def gulp(self, *args, **kwargs):
    exe = os.path.join(self.web_dir, 'node_modules', 'gulp', 'bin', 'gulp.js')
    return self.node(exe, *args, **kwargs)


def _subcommand_install():
  # Nothing to do, since toolchain is installed as a precondition to invoking
  # the subcommand!
  return 0


def _subcommand_presubmit(tc):
  # Run Gulp PRESUBMIT.
  tc.gulp('presubmit', cwd=tc.apps_dir)
  return 0


def _subcommand_build(tc, build_dir, apps=None):
  # Build requested apps.
  if not apps:
    # Get all apps with a gulpfile
    apps = [app for app in os.listdir(tc.apps_dir)
            if os.path.isfile(os.path.join(tc.apps_dir, app, 'gulpfile.js'))]

  for app in apps:
    LOGGER.info('Building app [%s] => [%s]', app, build_dir)
    tc.gulp('--out', build_dir,
        cwd=os.path.join(tc.apps_dir, app))
  return 0


def _subcommand_gulp(tc, gulp_args, app=None):
  app_dir = tc.apps_dir
  if app:
    app_dir = os.path.join(app_dir, app)
    if not os.path.isfile(os.path.join(app_dir, 'gulpfile.js')):
      raise ValueError('[%s] is not a valid application' % (app,))
  tc.gulp(*gulp_args, cwd=app_dir)
  return 0


def main(args):
  parser = argparse.ArgumentParser()
  parser.add_argument('-v', '--verbose', action='count', default=0,
      help='Increase verbosity.')
  parser.add_argument('-i', '--force-install', action='store_true',
      help='Install NPM/Bower files even if they are considered up-to-date.')

  subparser = parser.add_subparsers()

  # Subcommand: install
  subcommand = subparser.add_parser('install',
      help='Install toolchain and exit.')
  subcommand.set_defaults(func=lambda _tc, _args: _subcommand_install())

  # Subcommand: presubmit
  subcommand = subparser.add_parser('presubmit',
      help='Run web presubmit verification.')
  subcommand.set_defaults(func=lambda tc, _args: _subcommand_presubmit(tc))

  # Subcommand: build
  subcommand = subparser.add_parser('build', help='Build web apps.')
  subcommand.set_defaults(func=lambda tc, args:
      _subcommand_build(tc, args.build_dir, args.apps))
  subcommand.add_argument('apps', nargs='*',
      help='Specific apps to build. If none are specified, build all apps.')
  subcommand.add_argument('--build-dir', default=_WEB_BUILD_PATH,
      help='Path to the output build directory. Apps will be written to a '
           '"dist" folder under this path.')

  # Subcommand: gulp
  subcommand = subparser.add_parser('gulp',
      help='Run a global Gulp.js target.')
  subcommand.set_defaults(func=lambda tc, args:
      _subcommand_gulp(tc, args.gulp_args))
  subcommand.add_argument('gulp_args', nargs='*',
      help='Arguments to pass to Gulp.js.')

  # Subcommand: gulp-app
  subcommand = subparser.add_parser('gulp-app',
      help='Run Gulp.js for the specified web app.')
  subcommand.set_defaults(func=lambda tc, args:
      _subcommand_gulp(tc, args.gulp_args, app=args.app))
  subcommand.add_argument('app',
      help='Web app name')
  subcommand.add_argument('gulp_args', nargs='*',
      help='Arguments to pass to Gulp.js.')

  args = parser.parse_args(args)

  # Set logging level.
  if args.verbose > 0:
    LOGGER.setLevel(logging.DEBUG)

  # Build our generated web content.
  tc = Toolchain.initialize(_LUCI_GO_ROOT, force=args.force_install)
  return args.func(tc, args)


if __name__ == '__main__':
  logging.basicConfig(level=logging.INFO)
  sys.exit(main(sys.argv[1:]))
