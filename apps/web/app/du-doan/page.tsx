import { Footer, PageTitle } from "../ui";
import { PredictionStudio } from "./studio";

export default function PredictionsPage() {
  return (
    <>
      <main className="wrap predictions-page">
        <PageTitle
          eyebrow="PREDICTION & QUIZ"
          title="Thử tài hiểu thể thao"
          description="Dự đoán và câu hỏi kiến thức chỉ để tích điểm, không tiền thật, không odds, không cá cược."
        />
        <PredictionStudio />
      </main>
      <Footer />
    </>
  );
}
