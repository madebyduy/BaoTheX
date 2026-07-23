"use client";

import { useState } from "react";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

export function CheckoutButton({ price }: { price: number }) {
  const [state, setState] = useState("");
  async function checkout() {
    setState("Đang tạo phiên thanh toán…");
    const response = await fetch(`${API}/api/v1/premium/checkout`, {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) {
      setState(response.status === 401 ? "Bạn cần đăng nhập trước." : "Chưa thể khởi tạo SePay.");
      return;
    }
    const json = await response.json();
    const data = json.data ?? json;
    const form = document.createElement("form");
    form.method = "POST";
    form.action = data.action;
    for (const [name, value] of Object.entries(data.fields as Record<string, string>)) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = name;
      input.value = value;
      form.appendChild(input);
    }
    document.body.appendChild(form);
    form.submit();
  }
  return (
    <div>
      <button className="btn ember premium-cta" type="button" onClick={checkout}>
        Nâng cấp Premium · {new Intl.NumberFormat("vi-VN").format(price)}đ/tháng
      </button>
      {state ? <p className="settings-message">{state}</p> : null}
    </div>
  );
}
