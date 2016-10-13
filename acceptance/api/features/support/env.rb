require 'cucumber-api'

require 'fileutils'

def combineAllDataFiles(dir)
  data = "{"
  Dir.foreach(dir) do |file|
    next if file == '.' || file == '..'
    begin
      original = File.read(File.join(dir, file))
    rescue => err
      printf "ERROR: Dataset file %s could not be read: %s\n", file, err.message
      exit 1
    end
    data << removeWhitespaceAndOuterBrackets(original)
    data << ",\n\n"
  end
  data = data.rstrip.chop
  data << "\n}"
  return data
end

def removeWhitespaceAndOuterBrackets(text)
  text = text.strip
  text = (text[1..-2]).rstrip
  return text
end

def parseJson(data)
  begin
    data = JSON.parse(data)
  rescue => err
    printf "ERROR: Dataset file could not be parsed: %s\n", err.message
    exit 1
  end
  return data
end

# Loops through the items in the array and returns the
# item that matches the given property or nil.
def findArrayMatch(array, property, match)
  array.each do |item|
    return item if item[property] == match
  end
  return nil
end

def getTableValue(valueOrTableUrl)
  if valueOrTableUrl.start_with?("table://") == false
    return valueOrTableUrl
  end
  parsedUrl = valueOrTableUrl.split(/\W+/)
  if parsedUrl.size != 4
    raise(ArgumentError.new('Invalid URL'))
  end

  tableType = parsedUrl[1]
  tableName = parsedUrl[2]
  propertyName = parsedUrl[3]
  if PARSED_DATA[tableType].nil?
    raise(ArgumentError.new('Invalid table type'))
  elsif PARSED_DATA[tableType][tableName].nil?
    raise(ArgumentError.new('Invalid table name'))
  elsif PARSED_DATA[tableType][tableName][propertyName].nil?
    raise(ArgumentError.new('Invalid property name'))
  else
    data = PARSED_DATA[tableType][tableName][propertyName]
    if data.to_s.include? "%{local_ip}"
      data.sub! "%{local_ip}", HOST_IP
    end
    if data.to_s.include? "%{target_host}"
      data.sub! "%{target_host}", TARGET_HOST
    end
    return data
  end
end



printf "Setting up acceptance test env\n"

printf "Using app_host=%s\n", ENV["APPLICATION_URL"]
printf "Using userid=%s\n", ENV["APPLICATION_USERID"]

dataset_dir = File.join(ENV["DATASET_DIR"], ENV["DATASET"])
if !Dir.exists?(dataset_dir) || Dir.entries(dataset_dir).size <= 2
  printf "ERROR: DATASET_DIR is not defined; check cucumber.yml\n"
  exit 1
end

data = combineAllDataFiles(dataset_dir)
PARSED_DATA = parseJson(data)

template_dir = File.join(ENV["DATASET_DIR"], "/testsvc")
if !Dir.exists?(template_dir) || Dir.entries(template_dir).size <= 2
  printf "ERROR: #{template_dir} is not defined\n"
  exit 1
end
TEMPLATE_DIR = template_dir

printf "Using dataset directory=%s\n", ENV["DATASET_DIR"]
printf "Using dataset=%s\n", ENV["DATASET"]
printf "Using template directory=%s\n", template_dir


HOST_IP = ENV["HOST_IP"]
TARGET_HOST = ENV["TARGET_HOST"]
printf "Using HOST_IP=%s\n", HOST_IP
printf "Using TARGET_HOST=%s\n", TARGET_HOST

