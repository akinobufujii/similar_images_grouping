# similar_images_grouping
* Similar images under the specified directory Group similar images together.
* It's fast because it runs in parallel.
* See the article below for details.
  * [Goで「どの画像が似てるか」をグルーピングするツールを作った](https://zenn.dev/akinobufujii/articles/6dee09b659ca8c)

## Getting Started

### Prerequisites
* Requires `Go 1.18` or higher.

### Installing

```sh
git clone https://github.com/akinobufujii/similar_images_grouping.git
cd /path/to/similar_images_grouping
go install .
```

```sh
# Usage
similar_images_grouping -help

# Grouping similar images from Any Directory
# output result 'similar_groups.json'
similar_images_grouping -root="/path/to/any"
```

## Licence
MIT License - see the [LICENSE](LICENSE) file for details

