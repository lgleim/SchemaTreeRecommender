<?php

namespace PropertySuggester\UpdateTable;

use Wikimedia\Rdbms\ILBFactory;

/**
 * Context for importing data from a csv file to a db table using a Importer strategy
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
class ImportContext {

	/**
	 * file system path to the CSV to load data from
	 * @var string
	 */
	private $csvFilePath = "";

	/**
	 * delimiter used in csv file
	 * @var string
	 */
	private $csvDelimiter = ",";

	/**
	 * table name of the table to import to
	 * @var string
	 */
	private $targetTableName = "";

	/**
	 * @var ILBFactory|null
	 */
	private $lbFactory = null;

	/**
	 * @var int
	 */
	private $batchSize;

	/**
	 * @var boolean
	 */
	private $quiet;

	/**
	 * @return string
	 */
	public function getCsvDelimiter() {
		return $this->csvDelimiter;
	}

	/**
	 * @param string $csvDelimiter
	 */
	public function setCsvDelimiter( $csvDelimiter ) {
		$this->csvDelimiter = $csvDelimiter;
	}

	/**
	 * @return ILBFactory|null
	 */
	public function getLbFactory() {
		return $this->lbFactory;
	}

	public function setLbFactory( ILBFactory $lbFactory ) {
		$this->lbFactory = $lbFactory;
	}

	/**
	 * @return string
	 */
	public function getTargetTableName() {
		return $this->targetTableName;
	}

	/**
	 * @param string $tableName
	 */
	public function setTargetTableName( $tableName ) {
		$this->targetTableName = $tableName;
	}

	/**
	 * @return string
	 */
	public function getCsvFilePath() {
		return $this->csvFilePath;
	}

	/**
	 * @param string $fullPath
	 */
	public function setCsvFilePath( $fullPath ) {
		$this->csvFilePath = $fullPath;
	}

	/**
	 * @return int
	 */
	public function getBatchSize() {
		return $this->batchSize;
	}

	/**
	 * @param int $batchSize
	 */
	public function setBatchSize( $batchSize ) {
		$this->batchSize = $batchSize;
	}

	/**
	 * @return bool
	 */
	public function isQuiet() {
		return $this->quiet;
	}

	/**
	 * @param bool $quiet
	 */
	public function setQuiet( $quiet ) {
		$this->quiet = $quiet;
	}

}
