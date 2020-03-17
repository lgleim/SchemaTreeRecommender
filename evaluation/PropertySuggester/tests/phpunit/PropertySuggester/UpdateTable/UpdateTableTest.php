<?php

namespace PropertySuggester\UpdateTable;

use MediaWikiTestCase;
use PropertySuggester\Maintenance\UpdateTable;

/**
 * @covers \PropertySuggester\maintenance\UpdateTable
 * @covers \PropertySuggester\UpdateTable\Importer\BasicImporter
 * @covers \PropertySuggester\UpdateTable\ImportContext
 *
 * @group PropertySuggester
 * @group Database
 * @group medium
 */
class UpdateTableTest extends MediaWikiTestCase {

	/**
	 * @var string
	 */
	private $testfilename;

	/**
	 * @var string[]
	 */
	private $rowHeader = [ 'pid1', 'qid1', 'pid2', 'count', 'probability', 'context' ];

	public function setUp() {
		parent::setUp();

		$this->tablesUsed[] = 'wbs_propertypairs';
		$this->testfilename = sys_get_temp_dir() . '/_temp_test_csv_file.csv';
	}

	public function getRows() {
		$rows1 = [
			[ 1, 0, 2, 100, 0.1, 'item' ],
			[ 1, 0, 3, 50, 0.05, 'item' ],
			[ 2, 0, 3, 100, 0.1, 'item' ],
			[ 2, 0, 4, 200, 0.2, 'item' ],
			[ 3, 0, 1, 123, 0.5, 'item' ],
		];

		$rows2 = [];
		for ( $i = 0; $i < 1100; $i++ ) {
			$rows2[] = [ $i, 0, 2, 100, 0.1, 'item' ];
		}

		return [
			[ $rows1 ],
			[ $rows2 ],
		];
	}

	/**
	 * @dataProvider getRows
	 */
	public function testRewriteNativeStrategy( array $rows ) {
		$args = [ 'file' => $this->testfilename, 'quiet' => true, 'use-loaddata' => true ];
		$this->runScriptAndAssert( $args, $rows );
	}

	/**
	 * @dataProvider getRows
	 */
	public function testRewriteWithSQLInserts( array $rows ) {
		$args = [ 'file' => $this->testfilename, 'quiet' => true ];
		$this->runScriptAndAssert( $args, $rows );
	}

	private function runScriptAndAssert( array $args, array $rows ) {
		$this->setupData( $rows );
		$maintenanceScript = new UpdateTable();
		$maintenanceScript->loadParamsAndArgs( null, $args, null );
		$maintenanceScript->execute();
		if ( count( $rows ) < 100 ) {
			$this->assertSelect(
				'wbs_propertypairs',
				[ 'pid1', 'qid1', 'pid2', 'count', 'probability', 'context' ],
				[],
				$rows
			);
		} else {
			// assertSelect is too slow to compare 1100 rows... just check the size
			$this->assertSelect(
				'wbs_propertypairs',
				[ 'count' => 'count(*)' ],
				[],
				[ [ count( $rows ) ] ]
			);
		}
	}

	private function setupData( array $rows ) {
		$fhandle = fopen( $this->testfilename, 'w' );
		fputcsv( $fhandle, $this->rowHeader, ',' );
		foreach ( $rows as $row ) {
			fputcsv( $fhandle, $row, ',' );
		}
		fclose( $fhandle );
	}

	public function tearDown() {
		if ( file_exists( $this->testfilename ) ) {
			unlink( $this->testfilename );
		}
		parent::tearDown();
	}

}
