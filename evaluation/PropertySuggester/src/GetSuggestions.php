<?php

namespace PropertySuggester;

use ApiBase;
use ApiMain;
use ApiResult;
use DerivativeRequest;
use InvalidArgumentException;
use MediaWiki\MediaWikiServices;
use PropertySuggester\Suggesters\SimpleSuggester;
use PropertySuggester\Suggesters\SuggesterEngine;
use Wikibase\DataModel\Entity\Property;
use Wikibase\DataModel\Services\Lookup\EntityLookup;
use Wikibase\Lib\Store\EntityTitleLookup;
use Wikibase\Repo\Api\EntitySearchHelper;
use Wikibase\Repo\Api\TypeDispatchingEntitySearchHelper;
use Wikibase\Repo\WikibaseRepo;
use Wikibase\TermIndex;

/**
 * API module to get property suggestions.
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
class GetSuggestions extends ApiBase {

	/**
	 * @var EntityLookup
	 */
	private $entityLookup;

	/**
	 * @var EntityTitleLookup
	 */
	private $entityTitleLookup;

	/**
	 * @var string[]
	 */
	private $languageCodes;

	/**
	 * @var SuggesterEngine
	 */
	private $suggester;

	/**
	 * @var TermIndex
	 */
	private $termIndex;

	/**
	 * @var SuggesterParamsParser
	 */
	private $paramsParser;

	/**
	 * @var EntitySearchHelper
	 */
	private $entitySearchHelper;

	/**
	 * @param ApiMain $main
	 * @param string $name
	 * @param string $prefix
	 */
	public function __construct( ApiMain $main, $name, $prefix = '' ) {
		parent::__construct( $main, $name, $prefix );
		global $wgPropertySuggesterDeprecatedIds;
		global $wgPropertySuggesterMinProbability;
		global $wgPropertySuggesterClassifyingPropertyIds;
		global $wgPropertySuggesterInitialSuggestions;

		$wikibaseRepo = WikibaseRepo::getDefaultInstance();
		$store = $wikibaseRepo->getStore();
		$lb = MediaWikiServices::getInstance()->getDBLoadBalancer();

		$this->termIndex = $store->getTermIndex();
		$this->entitySearchHelper = new TypeDispatchingEntitySearchHelper(
			$wikibaseRepo->getEntitySearchHelperCallbacks(),
			$main->getRequest()
		);
		$this->entityLookup = $store->getEntityLookup();
		$this->entityTitleLookup = $wikibaseRepo->getEntityTitleLookup();
		$this->languageCodes = $wikibaseRepo->getTermsLanguages()->getLanguages();

		$this->suggester = new SimpleSuggester( $lb );
		$this->suggester->setDeprecatedPropertyIds( $wgPropertySuggesterDeprecatedIds );
		$this->suggester->setClassifyingPropertyIds( $wgPropertySuggesterClassifyingPropertyIds );
		$this->suggester->setInitialSuggestions( $wgPropertySuggesterInitialSuggestions );

		$this->paramsParser = new SuggesterParamsParser( 500, $wgPropertySuggesterMinProbability );
	}

	/**
	 * @see ApiBase::execute()
	 */
	public function execute() {
		$extracted = $this->extractRequestParams();
		$params = $this->paramsParser->parseAndValidate( $extracted );

		$suggestionGenerator = new SuggestionGenerator(
			$this->entityLookup,
			$this->entitySearchHelper,
			$this->suggester
		);

		$suggest = SuggesterEngine::SUGGEST_NEW;
		if ( $params->include === 'all' ) {
			$suggest = SuggesterEngine::SUGGEST_ALL;
		}
		if ( $params->entity !== null ) {
			try {
				$suggestions = $suggestionGenerator->generateSuggestionsByItem(
					$params->entity,
					$params->suggesterLimit,
					$params->minProbability,
					$params->context,
					$suggest
				);
			} catch ( InvalidArgumentException $ex ) {
				$this->dieWithException( $ex );
			}
		} else {
			$suggestions = $suggestionGenerator->generateSuggestionsByPropertyList(
				$params->properties,
				$params->suggesterLimit,
				$params->minProbability,
				$params->context,
				$suggest
			);
		}

		$suggestions = $suggestionGenerator->filterSuggestions(
			$suggestions,
			$params->search,
			$params->language,
			$params->resultSize
		);

		// Build result array
		$resultBuilder = new ResultBuilder(
			$this->getResult(),
			$this->termIndex,
			$this->entityTitleLookup,
			$params->search
		);

		$entries = $resultBuilder->createResultArray( $suggestions, $params->language );

		// merge with search result if possible and necessary
		if ( count( $entries ) < $params->resultSize && $params->search !== '' ) {
			$searchResult = $this->querySearchApi( $params->resultSize, $params->search, $params->language );
			$entries = $resultBuilder->mergeWithTraditionalSearchResults( $entries, $searchResult, $params->resultSize );
		}

		// Define Result
		$slicedEntries = array_slice( $entries, $params->continue, $params->limit );
		ApiResult::setIndexedTagName( $slicedEntries, 'search' );
		$this->getResult()->addValue( null, 'search', $slicedEntries );

		$this->getResult()->addValue( null, 'success', 1 );
		if ( count( $entries ) >= $params->resultSize ) {
			$this->getResult()->addValue( null, 'search-continue', $params->resultSize );
		}
		$this->getResult()->addValue( 'searchinfo', 'search', $params->search );
	}

	/**
	 * @param int $resultSize
	 * @param string $search
	 * @param string $language
	 * @return array[]
	 */
	private function querySearchApi( $resultSize, $search, $language ) {
		$searchEntitiesParameters = new DerivativeRequest(
			$this->getRequest(),
			[
				'limit' => $resultSize + 1,
				'continue' => 0,
				'search' => $search,
				'action' => 'wbsearchentities',
				'language' => $language,
				'uselang' => $language,
				'type' => Property::ENTITY_TYPE
			]
		);

		$api = new ApiMain( $searchEntitiesParameters );
		$api->execute();

		$apiResult = $api->getResult()->getResultData(
			null,
			[
				'BC' => [],
				'Types' => [],
				'Strip' => 'all'
			]
		);

		return $apiResult['search'];
	}

	/**
	 * @inheritDoc
	 */
	public function getAllowedParams() {
		return [
			'entity' => [
				ApiBase::PARAM_TYPE => 'string',
			],
			'properties' => [
				ApiBase::PARAM_TYPE => 'string',
				ApiBase::PARAM_ISMULTI => true
			],
			'limit' => [
				ApiBase::PARAM_TYPE => 'limit',
				ApiBase::PARAM_DFLT => 7,
				ApiBase::PARAM_MAX => ApiBase::LIMIT_SML1,
				ApiBase::PARAM_MAX2 => ApiBase::LIMIT_SML2,
				ApiBase::PARAM_MIN => 0,
			],
			'continue' => null,
			'language' => [
				ApiBase::PARAM_TYPE => $this->languageCodes,
				ApiBase::PARAM_DFLT => $this->getContext()->getLanguage()->getCode(),
			],
			'context' => [
				ApiBase::PARAM_TYPE => [ 'item', 'qualifier', 'reference' ],
				ApiBase::PARAM_DFLT => 'item',
			],
			'include' => [
				ApiBase::PARAM_TYPE => [ '', 'all' ],
				ApiBase::PARAM_DFLT => '',
			],
			'search' => [
				ApiBase::PARAM_TYPE => 'string',
				ApiBase::PARAM_DFLT => '',
			],
		];
	}

	/**
	 * @inheritDoc
	 */
	public function getExamplesMessages() {
		return [
			'action=wbsgetsuggestions&entity=Q4'
			=> 'apihelp-wbsgetsuggestions-example-1',
			'action=wbsgetsuggestions&entity=Q4&continue=10&limit=5'
			=> 'apihelp-wbsgetsuggestions-example-2',
			'action=wbsgetsuggestions&properties=P31|P21'
			=> 'apihelp-wbsgetsuggestions-example-3',
			'action=wbsgetsuggestions&properties=P21&context=qualifier'
			=> 'apihelp-wbsgetsuggestions-example-4',
			'action=wbsgetsuggestions&properties=P21&context=reference'
			=> 'apihelp-wbsgetsuggestions-example-5'
		];
	}

}
