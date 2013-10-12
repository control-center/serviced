-- MySQL dump 10.13  Distrib 5.5.32, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: q
-- ------------------------------------------------------
-- Server version	5.5.32-0ubuntu0.13.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `container`
--

DROP TABLE IF EXISTS `container`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `container` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `docker_id` varchar(36) NOT NULL,
  `host_id` varchar(36) NOT NULL,
  `ip_addr` varchar(36) NOT NULL,
  `launched_at` datetime NOT NULL,
  `terminted_at` datetime DEFAULT NULL,
  `service_id` varchar(36) DEFAULT NULL,
  `image_id` varchar(36) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `docker_id_UNIQUE` (`docker_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `container`
--

LOCK TABLES `container` WRITE;
/*!40000 ALTER TABLE `container` DISABLE KEYS */;
/*!40000 ALTER TABLE `container` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `host`
--

DROP TABLE IF EXISTS `host`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host` (
  `id` char(36) NOT NULL,
  `resource_pool_id` char(45) NOT NULL,
  `name` varchar(45) NOT NULL,
  `ip_addr` varchar(30) NOT NULL,
  `private_network` varchar(45) NOT NULL,
  `cores` int(11) NOT NULL,
  `memory` bigint(20) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `hostname_UNIQUE` (`name`),
  KEY `resource_pool` (`resource_pool_id`),
  KEY `fk_resource_pool` (`resource_pool_id`),
  CONSTRAINT `fk_resource_pool` FOREIGN KEY (`resource_pool_id`) REFERENCES `resource_pool` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `host`
--

LOCK TABLES `host` WRITE;
/*!40000 ALTER TABLE `host` DISABLE KEYS */;
/*!40000 ALTER TABLE `host` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `image`
--

DROP TABLE IF EXISTS `image`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `image` (
  `name` varchar(60) NOT NULL,
  `dockerfile` mediumtext NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `image`
--

LOCK TABLES `image` WRITE;
/*!40000 ALTER TABLE `image` DISABLE KEYS */;
/*!40000 ALTER TABLE `image` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `resource_pool`
--

DROP TABLE IF EXISTS `resource_pool`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `resource_pool` (
  `id` char(45) NOT NULL,
  `cores` int(11) DEFAULT '0',
  `memory` int(11) DEFAULT '0',
  `priority` int(11) DEFAULT '0',
  `parent_id` char(45) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `resource_pool`
--

LOCK TABLES `resource_pool` WRITE;
/*!40000 ALTER TABLE `resource_pool` DISABLE KEYS */;
/*!40000 ALTER TABLE `resource_pool` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service`
--

DROP TABLE IF EXISTS `service`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `service` (
  `id` char(36) NOT NULL DEFAULT '',
  `name` varchar(45) NOT NULL,
  `startup` text NOT NULL,
  `image_id` varchar(50) NOT NULL,
  `resource_pool_id` char(36) NOT NULL,
  `instances` int(11) NOT NULL,
  `description` varchar(255) NOT NULL,
  `desired_state` int(11) NOT NULL,
  `parent_service_id` char(36) NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_pool` (`resource_pool_id`),
  CONSTRAINT `fk_pool` FOREIGN KEY (`resource_pool_id`) REFERENCES `resource_pool` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service`
--

LOCK TABLES `service` WRITE;
/*!40000 ALTER TABLE `service` DISABLE KEYS */;
/*!40000 ALTER TABLE `service` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_endpoint`
--

DROP TABLE IF EXISTS `service_endpoint`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `service_endpoint` (
  `service_id` char(36) NOT NULL,
  `port` int(11) NOT NULL,
  `protocol` enum('tcp','udp') NOT NULL,
  `application` varchar(45) DEFAULT NULL,
  `purpose` enum('local','remote') NOT NULL,
  PRIMARY KEY (`service_id`,`port`,`protocol`),
  KEY `fk_port_1` (`service_id`),
  CONSTRAINT `fk_port_1` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_endpoint`
--

LOCK TABLES `service_endpoint` WRITE;
/*!40000 ALTER TABLE `service_endpoint` DISABLE KEYS */;
/*!40000 ALTER TABLE `service_endpoint` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_state`
--

DROP TABLE IF EXISTS `service_state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `service_state` (
  `id` char(36) NOT NULL,
  `service_id` varchar(36) NOT NULL,
  `terminated_at` datetime NOT NULL,
  `started_at` datetime NOT NULL,
  `scheduled_at` datetime NOT NULL,
  `host_id` varchar(36) NOT NULL,
  `docker_id` varchar(45) NOT NULL DEFAULT '',
  `private_ip` varchar(45) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `servicehost` (`service_id`,`host_id`),
  KEY `hostservice` (`host_id`,`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_state`
--

LOCK TABLES `service_state` WRITE;
/*!40000 ALTER TABLE `service_state` DISABLE KEYS */;
/*!40000 ALTER TABLE `service_state` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = '' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`root`@`localhost`*/ /*!50003 TRIGGER service_state_before BEFORE INSERT ON service_state FOR EACH ROW 
BEGIN
  IF EXISTS (
    SELECT * FROM service_state AS s 
    WHERE s.service_id = NEW.service_id 
        AND terminated_at IS NULL
   )
    THEN
        SIGNAL sqlstate '45001' set message_text = "Can not start non-terminated service";
    END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `service_state_endpoint`
--

DROP TABLE IF EXISTS `service_state_endpoint`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `service_state_endpoint` (
  `service_state_id` char(36) NOT NULL,
  `port` int(11) NOT NULL,
  `protocol` enum('tcp','udp') NOT NULL,
  `external_port` int(11) DEFAULT NULL,
  PRIMARY KEY (`service_state_id`,`port`,`protocol`),
  KEY `fk_service_state_endpoint_1` (`service_state_id`),
  CONSTRAINT `fk_service_state_endpoint_1` FOREIGN KEY (`service_state_id`) REFERENCES `service_state` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_state_endpoint`
--

LOCK TABLES `service_state_endpoint` WRITE;
/*!40000 ALTER TABLE `service_state_endpoint` DISABLE KEYS */;
/*!40000 ALTER TABLE `service_state_endpoint` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_template`
--

DROP TABLE IF EXISTS `service_template`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `service_template` (
  `id` char(36) NOT NULL,
  `name` varchar(45) NOT NULL,
  `description` varchar(255) NOT NULL,
  `data` text NOT NULL,
  `api_version` int(11) NOT NULL DEFAULT '0',
  `template_version` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

CREATE TABLE service_deployment
(
    `id` char(36) NOT NULL 
    ,PRIMARY KEY (id)
    ,`service_template_id` varchar(36) NOT NULL 
    ,`service_id` varchar(36) NOT NULL 
    ,`deployed_at` datetime NOT NULL 
)
ENGINE=INNODB
--
-- Dumping data for table `service_template`
--

LOCK TABLES `service_template` WRITE;
/*!40000 ALTER TABLE `service_template` DISABLE KEYS */;
/*!40000 ALTER TABLE `service_template` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2013-10-07 16:05:03
