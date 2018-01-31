# frozen_string_literal: true

require "spec_helper"

RSpec.describe "/instances" do
  def create_unready_image
    JSON.parse(
      post(
        "/images",
        data: {
          type: "images",
          attributes: {
            backed_up_at: Time.utc(2016, 1, 2, 3, 4, 5).iso8601,
            anonymisation_script: "CREATE TABLE foo (id serial);",
          },
        },
      ),
    )["data"]["id"]
  end
  # rubocop:enable Metrics/MethodLength

  def create_ready_image
    image_id = create_unready_image

    client.store_file("/tmp/db.tar", File.read("spec/fixtures/db.tar"))
    client.exec(["mv", "/tmp/db.tar", "/draupnir/image_uploads/#{image_id}/db.tar"])
    post("/images/#{image_id}/done", {})

    image_id
  end

  def create_instance(image_id)
    JSON.parse(
      post(
        "/instances",
        data: {
          type: "instances",
          attributes: {
            image_id: image_id,
          },
        },
      ),
    )["data"]["id"]
  end

  describe "POST /instances" do
    it "returns an error if given an unready image" do
      image_id = create_unready_image

      begin
        create_instance(image_id)
      rescue RestClient::UnprocessableEntity => e
        # TODO: fixture
        response = e.response
        expect(response.headers[:content_type]).to eq("application/json")
        expect(response.code).to eq(422)
        expect(JSON.parse(response.body)).to match(
          "id" => "unprocessable_entity",
          "status" => "422",
          "code" => "unprocessable_entity",
          "title" => "Image Not Ready",
          "detail" => "The specified image is not ready to be used",
          "source" => { "parameter" => "image_id" },
        )
      end
    end

    it "creates the instance if given a ready image" do
      image_id = create_ready_image

      response = post(
        "/instances",
        data: {
          type: "instances",
          attributes: {
            image_id: image_id,
          },
        },
      )
      expect(response.code).to eq(201)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(JSON.parse(response.body)).to match(
        "data" => {
          "id" => String,
          "type" => "instances",
          "attributes" => {
            "image_id" => image_id.to_i,
            "port" => Numeric,
            "created_at" => String,
            "updated_at" => String,
          },
        },
      )
    end
  end

  describe "GET /instances" do
    it "returns a JSON payload showing the instance" do
      image_id = create_ready_image
      instance_id = create_instance(image_id)

      response = get("/instances")
      expect(response.code).to eq(200)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(JSON.parse(response.body)).to match(
        "data" => [
          {
            "id" => instance_id,
            "type" => "instances",
            "attributes" => {
              "image_id" => image_id.to_i,
              "port" => Numeric,
              "updated_at" => String,
              "created_at" => String,
            },
          },
        ],
      )
    end
  end

  describe "GET /instances/:id" do
    it "shows the given instance" do
      image_id = create_ready_image
      instance_id = create_instance(image_id)

      response = get("/instances/#{instance_id}")
      expect(response.code).to eq(200)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(JSON.parse(response.body)).to match(
        "data" => {
          "id" => String,
          "type" => "instances",
          "attributes" => {
            "image_id" => image_id.to_i,
            "port" => Numeric,
            "updated_at" => String,
            "created_at" => String,
          },
        },
      )
    end
  end

  describe "DELETE /instances/:id" do
    it "deletes the instance and returns a 204" do
      image_id = create_ready_image
      instance_id = create_instance(image_id)

      response = delete("/instances/#{instance_id}")
      expect(response.code).to eq(204)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(response.body).to eq("")

      expect(JSON.parse(get("/instances").body)["data"]).to eq([])
    end
  end
end