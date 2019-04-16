python -c 'import sys; sys.path.insert(0, "/usr/share/yum-cli"); import cli; list = cli.YumBaseCli().returnPkgLists(["updates"]);res = ["{\"name\":\""+x.name+"\", \"version\":\""+x.evr+"\",\"arch\":\""+x.arch+"\",\"repository\":\""+x.repo.id+"\"}" for x in list.updates]; print "{\"updates\":["+",".join(res)+"]}"'



python -c 'import sys;sys.path.insert(0, "/usr/share/yum-cli");import cli;list = cli.YumBaseCli().returnPkgLists(["updates"]);print "".join(["{\"name\":\""+x.name+"\", \"available\":\""+x.evr+"\",\"arch\":\""+x.arch+"\",\"repo\":\""+x.repo.id+"\"}\n" for x in list.updates]);'