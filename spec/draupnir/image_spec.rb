# frozen_string_literal: true

require "spec_helper"

RSpec.describe "/images" do
  let(:post_payload) do
    {
      data: {
        type: "images",
        attributes: {
          backed_up_at: timestamp.iso8601,
          anonymisation_script: "CREATE DATABASE foo;",
        },
      },
    }
  end

  let(:timestamp) { Time.utc(2016, 1, 2, 3, 4, 5) }

  describe "POST /images" do
    let(:timestamp) { Time.utc(2016, 1, 2, 3, 4, 5) }

    context "with an invalid shared secret" do
      let(:secret) { "incorrectsecret" }

      it "returns an error" do
        response =
          begin
            post("/images", post_payload, authorization: "Bearer #{secret}")
          rescue RestClient::Unauthorized => e
            e.response
          end

        expect(response.code).to eq(401)
        expect(response.headers[:content_type]).to eq("application/json")
        expect(JSON.parse(response.body)).to match(
          "status" => "401",
          "id" => "unauthorized",
          "code" => "unauthorized",
          "title" => "Unauthorized",
          "detail" => "You do not have permission to view this resource",
          "source" => {},
        )
      end
    end

    context "with a valid shared secret" do
      let(:secret) { "thesharedsecret" }

      it "creates an image and serialises it as a response" do
        timestamp = Time.utc(2016, 1, 2, 3, 4, 5)
        response = post("/images", post_payload)

        expect(response.code).to eq(201)
        expect(response.headers[:content_type]).to eq("application/json")
        expect(JSON.parse(response.body)).to match(
          "data" => {
            "type" => "images",
            "id" => String,
            "attributes" => include(
              "backed_up_at" => timestamp.iso8601,
              "ready" => false,
              "created_at" => String,
              "updated_at" => String,
            ),
          },
        )
      end
    end
  end

  describe "GET /images" do
    before { post("/images", post_payload) }

    it "returns a JSON payload listing all the images" do
      response = get("/images")

      expect(response.code).to eq(200)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(JSON.parse(response.body)).to match(
        "data" => [
          {
            "type" => "images",
            "id" => String,
            "attributes" => include(
              "backed_up_at" => timestamp.iso8601,
              "ready" => false,
              "created_at" => String,
              "updated_at" => String,
            ),
          },
        ],
      )
    end
  end

  describe "GET /images/:id" do
    let!(:image_id) do
      JSON.parse(post("/images", post_payload))["data"]["id"]
    end

    it "returns a JSON payload showing the image" do
      response = get("/images/#{image_id}")

      expect(response.code).to eq(200)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(JSON.parse(response.body)).to match(
        "data" => {
          "type" => "images",
          "id" => String,
          "attributes" => include(
            "backed_up_at" => timestamp.iso8601,
            "ready" => false,
            "created_at" => String,
            "updated_at" => String,
          ),
        },
      )
    end
  end

  describe "DELETE /images/:id" do
    let!(:image_id) do
      JSON.parse(post("/images", post_payload))["data"]["id"]
    end

    it "deletes the image and returns a 204" do
      response = delete("/images/#{image_id}")

      expect(response.code).to eq(204)
      expect(response.headers[:content_type]).to eq("application/json")
      expect(response.body).to eq("")
    end
  end
end
