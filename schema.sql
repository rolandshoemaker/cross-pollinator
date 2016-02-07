CREATE TABLE `certificates` (
  `id` int NOT NULL AUTO_INCREMENT,
  `hash` binary(32) UNIQUE NOT NULL,
  `offset` bigint NOT NULL,
  `length` int NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `chains` (
  `id` int NOT NULL AUTO_INCREMENT,
  `hash` binary(32) UNIQUE NOT NULL,
  -- `certIDs` jsonb UNIQUE NOT NULL,
  `rootDN` varchar(255) NOT NULL,
  `entryType` tinyint(8) NOT NULL,
  `unparseableComponent` tinyint(1) NOT NULL,
  `logs` jsonb NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `logEntries` (
  `id` int NOT NULL AUTO_INCREMENT,
  `chainID` int NOT NULL,
  `entryNum` int NOT NULL,
  `logID` binary(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `entryNum_logID` (`entryNum`, `logID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
