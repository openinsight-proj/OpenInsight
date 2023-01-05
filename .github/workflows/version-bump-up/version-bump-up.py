import logging
import argparse
from github import Github
import os
from base64 import b64decode
import datetime
from packaging import version
from mergedeep import merge, Strategy
import re
from ruamel.yaml import load, dump, RoundTripLoader, RoundTripDumper

logging.basicConfig(level=logging.INFO, format='%(asctime)s %(name)s %(levelname)s %(message)s')
logger = logging.getLogger(__name__)
parser = argparse.ArgumentParser(description='version sync app')
parser.add_argument('--job', type=str, default='upstream', help='Choose a sync job: upstream, local. The default '
                                                                'value is upstream')
parser.add_argument('--gomodSyncStrategy', type=str, default='remain', help='Choose gomod version sync strategy for '
                                                                            'openinsight-distrubution manifest, '
                                                                            'available strategy: remain, sync. '
                                                                            'remain for not sync with upstream, '
                                                                            'sync for sync with upstream')
args = parser.parse_args()
token = os.getenv("MY_GITHUB_TOKEN")
if token is None:
    logger.error("need MY_GITHUB_TOKEN env")
    exit(1)
g = Github(token)
otelColRepo = g.get_repo("open-telemetry/opentelemetry-collector-releases")
openinsightRepo = g.get_repo("openinsight-proj/OpenInsight")

branchTemplate = 'bump_up_otelcol_contrib_to_{}'
cbilityFileNameTemplate = 'version_compatibility_{}.md'

prBodyTemplate = '''
This PR is auto create by a version bump up script, if there are other PRs create by the scripts, please approve the oldest one.
'''


def localJob():
    print("current not support")


def getRepoContents():
    upstreamMfest = b64decode(
        otelColRepo.get_contents("/distributions/otelcol-contrib/manifest.yaml").raw_data['content']).decode('utf-8')

    openinsightMfest = b64decode(
        openinsightRepo.get_contents("builder/otelcol-builder.yaml").raw_data['content']).decode('utf-8')
    openinsightDistruMfest = b64decode(
        openinsightRepo.get_contents("builder/openinsight-distrubution.yaml").raw_data['content']).decode('utf-8')
    openinsightReadme = b64decode(openinsightRepo.get_readme().raw_data['content']).decode('utf-8')

    year = datetime.datetime.now().year
    cbilityFile = cbilityFileNameTemplate.format(year)
    files = openinsightRepo.get_contents("docs")
    if cbilityFile in [f.name for f in files]:
        openinsightCbility = b64decode(
            openinsightRepo.get_contents("docs/version_compatibility_{}.md".format(year)).raw_data['content']).decode(
            'utf-8')
    else:
        openinsightCbility = b64decode(
            openinsightRepo.get_contents("docs/version_compatibility_{}.md".format(year - 1)).raw_data[
                'content']).decode('utf-8')

    return {
        'upstreamMfest': load(upstreamMfest, Loader=RoundTripLoader),
        'openinsightMfest': load(openinsightMfest, Loader=RoundTripLoader),
        'openinsightDistruMfest': load(openinsightDistruMfest, Loader=RoundTripLoader),
        'openinsightReadme': openinsightReadme,
        'openinsightCbility': openinsightCbility,
    }


def gomodSync(manifest, version):
    resource = ['extensions', 'exporters', 'processors', 'receivers']
    for key in manifest.keys():
        if key in resource:
            d = manifest[key]
            for value in d:
                gomod = value['gomod']
                strinfo = re.compile('v[0-9]+\.[0-9]+\.[0-9]+$')
                value['gomod'] = strinfo.sub(version, gomod)
    return manifest


def parseComponent(str):
    return str.split(' ')[0].split('/')[-1]


def updateManifest(upstreamMfest, openinsightDistruMfest):
    for k, v in openinsightDistruMfest.items():
        if k == 'dist':
            continue
        if k == 'replaces':
            continue
        for v2 in v:
            target = parseComponent(v2['gomod'])
            resource = upstreamMfest[k]
            for c in resource:
                if parseComponent(c['gomod']) == target:
                    resource.remove(c)
                    break

    newManifest = {}
    match args.gomodSyncStrategy:
        case 'remain':
            merge(newManifest, upstreamMfest, openinsightDistruMfest, strategy=Strategy.ADDITIVE)
        case 'sync':
            merge(newManifest, upstreamMfest, gomodSync(openinsightDistruMfest, upstreamMfest['dist']['version']),
                  strategy=Strategy.ADDITIVE)
        case _:
            logger.info("not support gomod sync strategy:{}".format(args.gomodSyncStrategy))
            exit(1)

    dist = """
    name: otelcol-contrib # the binary name. Optional.
    description: "OpenInsight. You know, OpenTelemetry Collector enhancement distribution" # a long name for the application. Optional.
    version: v0.63.0
    """
    distMap = load(dist, Loader=RoundTripLoader)
    distMap['version'] = upstreamMfest['dist']['version']
    newManifest['dist'] = distMap
    newManifest['dist'].yaml_set_start_comment("This file is auto-generated by .github/workflows/version-bump-up/version-bump-up.py.")
    return newManifest


def updateReadme(upstreamVersion, openinsightReadme, openinsightCbility):
    lines = openinsightCbility.split('\n')
    versionLine = lines[4]
    matched = re.search('(v[0-9]+\.[0-9]+\.[0-9]+)', versionLine)
    if matched is not None and len(matched.regs) == 2:
        localVersion = versionLine[matched.regs[0][0]:matched.regs[0][1]]
    latestVersionRow = '| {}              | v{}                  |'.format(localVersion, upstreamVersion)
    lines.insert(4, latestVersionRow)
    newCbility = '\n'.join(lines)

    del lines[:]
    openinsightReadme = openinsightReadme.split('\n')
    versionList = re.compile('^\| v[0-9]+\.[0-9]+\.[0-9]+\s+\| v[0-9]+\.[0-9]+\.[0-9]+\s+\|$')
    fileLink = re.compile('(docs/version_compatibility_[0-9]{4}.md)')
    for line in openinsightReadme:
        if re.search(versionList, line) is not None:
            lines.append(latestVersionRow)
            continue
        if re.search(fileLink, line) is not None:
            lines.append(
                'This table only show the latest version compatibility, more version compatibility please refer [version compatiblity list](docs/version_compatibility_{}.md)'.format(
                    datetime.datetime.today().year))
            continue
        lines.append(line)
    newReadme = '\n'.join(lines)

    return newReadme, newCbility


def pushToOpeninsight(newManifest, newReadme, newCbility):
    latestVersion = newManifest['dist']['version']
    openinsightRef = openinsightRepo.get_git_ref("heads")
    for v in openinsightRef.raw_data:
        if v['ref'] == 'refs/heads/main':
            latestSha = v['object']['sha']
    branchName = branchTemplate.format(latestVersion)
    ref = 'refs/heads/{}'.format(branchName)
    branch = openinsightRepo.create_git_ref(ref, latestSha)
    # sha = branch.raw_data['object']['sha']

    openinsightRepo.update_file('builder/otelcol-builder.yaml',
                                'update otelcol-builder.yaml to latest version',
                                dump(newManifest, Dumper=RoundTripDumper, width=float("inf")),
                                openinsightRepo.get_contents('builder/otelcol-builder.yaml', ref).sha,
                                branchName)

    openinsightRepo.update_file('README.md',
                                'update README.md',
                                newReadme,
                                openinsightRepo.get_contents('README.md', ref).sha,
                                branchName)

    files = openinsightRepo.get_contents("docs")
    year = datetime.datetime.now().year
    cbilityFile = cbilityFileNameTemplate.format(year)
    if cbilityFile in [f.name for f in files]:
        openinsightRepo.update_file('docs/{}'.format(cbilityFile),
                                    'update {}'.format(cbilityFile),
                                    newCbility,
                                    openinsightRepo.get_contents('docs/{}'.format(cbilityFile), ref).sha,
                                    branchName)
    else:
        openinsightRepo.create_file('docs/{}'.format(cbilityFile),
                                    'create {}'.format(cbilityFile),
                                    newCbility,
                                    branchName)
    openinsightRepo.create_pull('[chore] Bump up otel col contrib to v{}'.format(latestVersion), prBodyTemplate, 'main',
                                branchName, True)


def checkBranchs(latestVersion):
    branchName = branchTemplate.format(latestVersion)
    branches = list(openinsightRepo.get_branches())
    if branchName in [b.name for b in branches]:
        logger.error("There are already have a PR for This version")
        return False
    else:
        return True


def upstreamJob():
    repoContents = getRepoContents()
    upstreamVersion, openinsightVersion = repoContents['upstreamMfest']['dist']['version'], \
                                          repoContents['openinsightMfest']['dist']['version']
    logger.info('upstream verion:v{}, openinsightCbility version:v{}'.format(upstreamVersion, openinsightVersion))
    # if version.parse(upstreamVersion) > version.parse(openinsightVersion) and checkBranchs(upstreamVersion):
    if version.parse(upstreamVersion) > version.parse(openinsightVersion):

        logger.info("detect a new version and no upgrade PR for is version, do sync")
        newManifest = updateManifest(repoContents['upstreamMfest'], repoContents['openinsightDistruMfest'])
        newReadme, newCbility = updateReadme(upstreamVersion, repoContents['openinsightReadme'],
                                             repoContents['openinsightCbility'])
        pushToOpeninsight(newManifest, newReadme, newCbility)
        logger.info('success')


def main():
    job = args.job
    match job:
        case 'upstream':
            upstreamJob()
        case 'local':
            localJob()
        case _:
            logger.error("unsupported job")
            exit(1)


if __name__ == '__main__':
    main()
