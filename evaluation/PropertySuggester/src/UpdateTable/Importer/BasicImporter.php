<?php

namespace PropertySuggester\UpdateTable\Importer;

use UnexpectedValueException;
use PropertySuggester\UpdateTable\ImportContext;
use Wikimedia\Rdbms\IDatabase;
use Wikimedia\Rdbms\ILBFactory;

/**
 * A strategy which imports entries from a CSV file into a DB table. Used as fallback, when no
 * special import commands are supported by the DBMS.
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
class BasicImporter implements Importer {

	/**
	 * Import using SQL Insert
	 * @param ImportContext $importContext
	 * @return bool
	 */
	public function importFromCsvFileToDb( ImportContext $importContext ) {
		$fileHandle = fopen( $importContext->getCsvFilePath(), "r" );
		if ( $fileHandle == false ) {
			return false;
		}

		$lbFactory = $importContext->getLbFactory();
		$lb = $lbFactory->getMainLB();
		$db = $lb->getConnection( DB_MASTER );
		$this->doImport( $fileHandle, $lbFactory, $db, $importContext );
		$lb->reuseConnection( $db );

		fclose( $fileHandle );

		return true;
	}

	/**
	 * @param resource $fileHandle
	 * @param ILBFactory $lbFactory
	 * @param IDatabase $db
	 * @param ImportContext $importContext
	 * @throws UnexpectedValueException
	 * @suppress SecurityCheck-SQLInjection ImportContext::getTargetTableName is marked as unsafe
	 */
	private function doImport( $fileHandle, ILBFactory $lbFactory, IDatabase $db, ImportContext $importContext ) {
		$accumulator = [];
		$batchSize = $importContext->getBatchSize();
		$i = 0;
		$header = fgetcsv( $fileHandle, 0, $importContext->getCsvDelimiter() );
		$expectedHeader = [ 'pid1', 'qid1', 'pid2', 'count', 'probability', 'context' ];
		if ( $header != $expectedHeader ) {
			throw new UnexpectedValueException(
				"provided csv-file does not match the expected format:\n" . implode( ',', $expectedHeader )
			);
		}

		while ( true ) {
			$data = fgetcsv( $fileHandle, 0, $importContext->getCsvDelimiter() );

			if ( $data == false || ++$i % $batchSize == 0 ) {
				$db->commit( __METHOD__, 'flush' );
				$lbFactory->waitForReplication();
				$db->insert( $importContext->getTargetTableName(), $accumulator );
				if ( !$importContext->isQuiet() ) {
					print "$i rows inserted\n";
				}
				$accumulator = [];
				if ( $data == false ) {
					break;
				}
			}

			$qid1 = is_numeric( $data[1] ) ? $data[1] : 0;

			$accumulator[] = [
				'pid1' => $data[0],
				'qid1' => $qid1,
				'pid2' => $data[2],
				'count' => $data[3],
				'probability' => $data[4],
				'context' => $data[5],
			];
		}
	}

}
