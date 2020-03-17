<?php
/**
 * ----------------------------------------------------------------------------------------
 * This file is provided by the wikibase/wikibase docker image.
 * This file will be passed through envsubst which will replace "$" with "$".
 * If you want to change MediaWiki or Wikibase settings then either mount a file over this
 * template and or run a different entrypoint.
 * ----------------------------------------------------------------------------------------
 */

## Database settings
## Environment variables will be substituted in here.
$wgDBserver = "mysql.svc:3306";
$wgDBname = "my_wiki";
$wgDBuser = "wikiuser";
$wgDBpassword = "sqlpass";

## Logs
## Save these logs inside the container
$wgDebugLogGroups = [
	'resourceloader' => '/var/log/mediawiki/resourceloader.log',
	'exception' => '/var/log/mediawiki/exception.log',
	'error' => '/var/log/mediawiki/error.log',
];

## Site Settings
# TODO pass in the rest of this with env vars?
$wgShellLocale = "en_US.utf8";
$wgLanguageCode = "en";
$wgSitename = "wikibase-docker";
$wgMetaNamespace = "Project";
# Configured web paths & short URLs
# This allows use of the /wiki/* path
## https://www.mediawiki.org/wiki/Manual:Short_URL
$wgScriptPath = "/w";        // this should already have been configured this way
$wgArticlePath = "/wiki/$1";

#Set Secret
$wgSecretKey = "secretkey";

## RC Age
# https://www.mediawiki.org/wiki/Manual:
# Items in the recentchanges table are periodically purged; entries older than this many seconds will go.
# The query service (by default) loads data from recent changes
# Set this to 1 year to avoid any changes being removed from the RC table over a shorter period of time.
$wgRCMaxAge = 365 * 24 * 3600;

wfLoadSkin( 'Vector' );

## Wikibase
# Load Wikibase repo, client & lib with the example / default settings.
require_once "$IP/extensions/Wikibase/lib/WikibaseLib.php";
require_once "$IP/extensions/Wikibase/repo/Wikibase.php";
require_once "$IP/extensions/Wikibase/repo/ExampleSettings.php";
require_once "$IP/extensions/Wikibase/client/WikibaseClient.php";
require_once "$IP/extensions/Wikibase/client/ExampleSettings.php";
# OAuth
wfLoadExtension( 'OAuth' );
$wgGroupPermissions['sysop']['mwoauthproposeconsumer'] = true;
$wgGroupPermissions['sysop']['mwoauthmanageconsumer'] = true;
$wgGroupPermissions['sysop']['mwoauthviewprivate'] = true;
$wgGroupPermissions['sysop']['mwoauthupdateownconsumer'] = true;

# WikibaseImport
require_once "$IP/extensions/WikibaseImport/WikibaseImport.php";

# CirrusSearch
wfLoadExtension( 'Elastica' );
require_once "$IP/extensions/CirrusSearch/CirrusSearch.php";
$wgCirrusSearchServers = [ 'elasticsearch.svc' ];
$wgSearchType = 'CirrusSearch';
$wgCirrusSearchExtraIndexSettings['index.mapping.total_fields.limit'] = 5000;

# UniversalLanguageSelector
wfLoadExtension( 'UniversalLanguageSelector' );

# cldr
wfLoadExtension( 'cldr' );

# EntitySchema
wfLoadExtension( 'EntitySchema' );

# PropertySuggester
wfLoadExtension( 'PropertySuggester' );
#require_once "$IP/extensions/PropertySuggester/PropertySuggester.php";                                                                                                                                                                     
$wgPropertySuggesterMinProbability=0.0;

$wgMainCacheType = CACHE_NONE;
$wgCacheDirectory = false;
$defaultSuggestionLimit=1000;
$wgShowExceptionDetails = true;
